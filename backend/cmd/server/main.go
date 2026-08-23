package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"food-allergen-crosscontact-analyzer/backend/internal/router"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration_invalid", "error", err)
		os.Exit(1)
	}
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	database, err := repository.Open(cfg)
	if err != nil {
		logger.Error("database_open_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("database_close_failed", "error", err)
		}
	}()
	startup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if cfg.DBAutoMigrate {
		if err := database.Migrate(startup); err != nil {
			logger.Error("migration_failed", "error", err)
			os.Exit(1)
		}
	}
	if err := database.Seed(startup); err != nil {
		logger.Error("seed_failed", "error", err)
		os.Exit(1)
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
		logger.Error("assessment_service_failed", "error", err)
		os.Exit(1)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	handlers := router.Handlers{Support: handler.NewSupportHandler(supportService, validate), Profiles: handler.NewProfileHandler(profileService, validate), Routes: handler.NewRouteHandler(routeService, validate), Edges: handler.NewContactEdgeHandler(edgeService, validate), Assessments: handler.NewAssessmentHandler(assessmentService, validate)}
	engine := router.New(cfg, logger, database, supportService, handlers)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_started", "port", cfg.Port, "db_driver", cfg.DBDriver)
		serverErrors <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		logger.Info("shutdown_signal", "signal", sig.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
	}
	shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	if err := server.Shutdown(shutdown); err != nil {
		logger.Error("graceful_shutdown_failed", "error", err)
		_ = server.Close()
	}
	logger.Info("server_stopped")
}
