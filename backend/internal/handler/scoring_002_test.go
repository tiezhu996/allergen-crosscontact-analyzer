package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"food-allergen-crosscontact-analyzer/backend/internal/router"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func scoringEngine(t *testing.T) *gin.Engine {
	t.Helper()
	cfg := config.Config{
		Port:                "0",
		DBDriver:            "sqlite",
		DBDSN:               "file:scoring002?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
		DBAutoMigrate:       false,
		JWTSecret:           "scoring-002-jwt-secret-at-least-32-bytes",
		JWTExpiry:           time.Hour,
		CORSOrigins:         []string{"http://127.0.0.1:18525"},
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
	supportRepo := repository.NewSupportRepository(database.DB)
	profileRepo := repository.NewProfileRepository(database.DB)
	routeRepo := repository.NewRouteRepository(database.DB)
	edgeRepo := repository.NewContactEdgeRepository(database.DB)
	assessmentRepo := repository.NewAssessmentRepository(database.DB)
	supportService := service.NewSupportService(supportRepo, cfg)
	profileService := service.NewProfileService(profileRepo)
	routeService := service.NewRouteService(routeRepo, profileRepo)
	edgeService := service.NewContactEdgeService(edgeRepo, routeRepo)
	assessmentService, err := service.NewAssessmentService(assessmentRepo, routeRepo, profileRepo, edgeRepo, cfg)
	if err != nil {
		t.Fatalf("assessment service: %v", err)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
	handlers := router.Handlers{
		Support:     handler.NewSupportHandler(supportService, validate),
		Profiles:    handler.NewProfileHandler(profileService, validate),
		Routes:      handler.NewRouteHandler(routeService, validate),
		Edges:       handler.NewContactEdgeHandler(edgeService, validate),
		Assessments: handler.NewAssessmentHandler(assessmentService, validate),
	}
	return router.New(cfg, logger, database, supportService, handlers)
}

func scoringToken(t *testing.T, engine *gin.Engine) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"analyst","password":"Analyst#525"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if body.Data.Token == "" {
		t.Fatalf("empty token: %s", rec.Body.String())
	}
	return body.Data.Token
}

func TestMissingEdgeGetReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := scoringEngine(t)
	token := scoringToken(t, engine)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-edges/999999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET missing edge status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestMissingEdgeUpdateReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := scoringEngine(t)
	token := scoringToken(t, engine)
	payload := `{"contact_type":"sequence","shared_equipment":"Belt 9","cleaning_factor":0.5,"carryover_probability":0.5,"evidence_note":"Updated evidence for a missing edge.","enabled":true,"expected_version":1}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contact-edges/999999", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PUT missing edge status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}
