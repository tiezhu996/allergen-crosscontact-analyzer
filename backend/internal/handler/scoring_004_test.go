package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func newScoring004(t *testing.T, roles ...constants.Role) *gin.Engine {
	t.Helper()
	role := constants.RoleQualityAnalyst
	if len(roles) > 0 {
		role = roles[0]
	}
	cfg := config.Config{
		Port:                "0",
		DBDriver:            "sqlite",
		DBDSN:               "file:scoring004?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
		DBAutoMigrate:       false,
		JWTSecret:           "scoring-004-jwt-secret-at-least-32-bytes",
		JWTExpiry:           time.Hour,
		CORSOrigins:         []string{},
		MaxPropagationDepth: 12,
		Thresholds:          config.Thresholds{Medium: 0.12, High: 0.35, Critical: 0.65, Version: "2026.1"},
		RateLimitPerMinute:  1000,
		LogLevel:            "info",
	}
	database, err := repository.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := database.Seed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	profileRepo := repository.NewProfileRepository(database.DB)
	routeRepo := repository.NewRouteRepository(database.DB)
	edgeRepo := repository.NewContactEdgeRepository(database.DB)
	assessmentRepo := repository.NewAssessmentRepository(database.DB)
	assessmentService, err := service.NewAssessmentService(assessmentRepo, routeRepo, profileRepo, edgeRepo, cfg)
	if err != nil {
		t.Fatalf("assessment service: %v", err)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	h := handler.NewAssessmentHandler(assessmentService, validate)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.PrincipalKey, service.Principal{ID: 1, Username: "analyst", DisplayName: "分析员", Role: role})
		c.Next()
	})
	engine.POST("/matrix/compute", h.Preview)
	engine.GET("/assessments", h.List)
	engine.GET("/assessments/:id", h.Get)
	engine.POST("/assessments", h.Create)
	engine.POST("/assessments/:id/run", h.Run)
	engine.POST("/assessments/:id/review", h.Review)
	return engine
}

func doRequest(engine *gin.Engine, method, path, body string, cancelled bool) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cancelled {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func createAssessment(t *testing.T, engine *gin.Engine) uint {
	t.Helper()
	rec := doRequest(engine, http.MethodPost, "/assessments", `{"route_id":1}`, false)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create assessment: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	return body.Data.ID
}

func TestPreviewCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t)
	rec := doRequest(engine, http.MethodPost, "/matrix/compute", `{"route_id":1}`, true)
	if rec.Code == http.StatusOK {
		t.Fatalf("preview ignored a cancelled request context (status 200); cancellation must propagate")
	}
}

func TestCreateCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t)
	rec := doRequest(engine, http.MethodPost, "/assessments", `{"route_id":1}`, true)
	if rec.Code == http.StatusCreated {
		t.Fatalf("create ignored a cancelled request context (status 201); cancellation must propagate")
	}
}

func TestRunCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t)
	id := createAssessment(t, engine)
	rec := doRequest(engine, http.MethodPost, "/assessments/"+strconv.FormatUint(uint64(id), 10)+"/run", "", true)
	if rec.Code == http.StatusOK {
		t.Fatalf("run ignored a cancelled request context (status 200); cancellation must propagate")
	}
}

func TestReviewCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t, constants.RoleAdmin)
	id := createAssessment(t, engine)
	runRec := doRequest(engine, http.MethodPost, "/assessments/"+strconv.FormatUint(uint64(id), 10)+"/run", "", false)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run before review: %d %s", runRec.Code, runRec.Body.String())
	}
	rec := doRequest(engine, http.MethodPost, "/assessments/"+strconv.FormatUint(uint64(id), 10)+"/review", `{"decision":"accepted","reason":"Looks correct to me."}`, true)
	if rec.Code == http.StatusOK {
		t.Fatalf("review ignored a cancelled request context (status 200); cancellation must propagate")
	}
}

func TestListCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t)
	rec := doRequest(engine, http.MethodGet, "/assessments?page=1&page_size=20", "", true)
	if rec.Code == http.StatusOK {
		t.Fatalf("list ignored a cancelled request context (status 200); cancellation must propagate")
	}
}

func TestGetCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newScoring004(t)
	id := createAssessment(t, engine)
	rec := doRequest(engine, http.MethodGet, "/assessments/"+strconv.FormatUint(uint64(id), 10), "", true)
	if rec.Code == http.StatusOK {
		t.Fatalf("get ignored a cancelled request context (status 200); cancellation must propagate")
	}
}
