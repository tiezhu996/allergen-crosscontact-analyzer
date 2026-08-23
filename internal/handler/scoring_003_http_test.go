package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestProfileDetailUsageEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		Port:                "0",
		DBDriver:            "sqlite",
		DBDSN:               "file:scoring003http?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
		DBAutoMigrate:       false,
		JWTSecret:           "scoring-003-jwt-secret-at-least-32-bytes",
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
	engine := router.New(cfg, logger, database, supportService, handlers)

	// login as analyst
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"analyst","password":"Analyst#525"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	engine.ServeHTTP(loginRec, loginReq)
	var loginBody struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	// create an unused profile
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", strings.NewReader(`{"profile_code":"MAT-UNUSED-003","material_name":"Unused paste","allergens":["Tree Nut"],"source_type":"supplier_statement","profile_status":"active"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+loginBody.Data.Token)
	createRec := httptest.NewRecorder()
	engine.ServeHTTP(createRec, createReq)
	var created struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil || created.Data.ID == 0 {
		t.Fatalf("create profile failed: %d %s", createRec.Code, createRec.Body.String())
	}
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+strconv.FormatUint(uint64(created.Data.ID), 10), nil)
	getReq.Header.Set("Authorization", "Bearer "+loginBody.Data.Token)
	getRec := httptest.NewRecorder()
	engine.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get profile: %d %s", getRec.Code, getRec.Body.String())
	}
	var detail struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	usage, ok := detail.Data["used_by_routes"]
	if !ok {
		t.Fatal("detail missing used_by_routes")
	}
	switch v := usage.(type) {
	case []any:
		if v == nil {
			t.Fatal("used_by_routes is null; must be an empty array")
		}
	case nil:
		t.Fatal("used_by_routes is null; must be an empty array")
	default:
		t.Fatalf("used_by_routes has unexpected type %T", usage)
	}
}
