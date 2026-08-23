package router

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func TestReadyzCancelPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		Port:                "0",
		DBDriver:            "sqlite",
		DBDSN:               "file:scoring004ready?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
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
	handlers := Handlers{
		Support:     handler.NewSupportHandler(supportService, validate),
		Profiles:    handler.NewProfileHandler(profileService, validate),
		Routes:      handler.NewRouteHandler(routeService, validate),
		Edges:       handler.NewContactEdgeHandler(edgeService, validate),
		Assessments: handler.NewAssessmentHandler(assessmentService, validate),
	}
	engine := New(cfg, logger, database, supportService, handlers)
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx2)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz with cancelled request = %d, want 503 (request deadline must propagate)", rec.Code)
	}
}
