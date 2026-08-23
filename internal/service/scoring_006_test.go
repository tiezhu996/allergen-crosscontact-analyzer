package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"gorm.io/datatypes"
)

type scoring006Env struct {
	ctx       context.Context
	svc       *AssessmentService
	routeRepo repository.RouteRepository
	runRepo   repository.AssessmentRepository
	db        *repository.Database
}

func newScoring006(t *testing.T) *scoring006Env {
	t.Helper()
	cfg := config.Config{
		DBDriver:            "sqlite",
		DBDSN:               "file:scoring006?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
		JWTSecret:           "scoring-006-jwt-secret-at-least-32-bytes",
		JWTExpiry:           time.Hour,
		MaxPropagationDepth: 12,
		Thresholds:          config.Thresholds{Medium: 0.12, High: 0.35, Critical: 0.65, Version: "2026.1"},
		RateLimitPerMinute:  1000,
	}
	db, err := repository.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Seed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	routeRepo := repository.NewRouteRepository(db.DB)
	profileRepo := repository.NewProfileRepository(db.DB)
	edgeRepo := repository.NewContactEdgeRepository(db.DB)
	runRepo := repository.NewAssessmentRepository(db.DB)
	svc, err := NewAssessmentService(runRepo, routeRepo, profileRepo, edgeRepo, cfg)
	if err != nil {
		t.Fatalf("assessment service: %v", err)
	}
	return &scoring006Env{ctx: ctx, svc: svc, routeRepo: routeRepo, runRepo: runRepo, db: db}
}

func (e *scoring006Env) queueRun(t *testing.T, routeID uint, snapshotJSON []byte) uint {
	t.Helper()
	run := model.AssessmentRun{RouteID: routeID, AssessmentStatus: constants.AssessmentQueued, InputSnapshotJSON: datatypes.JSON(snapshotJSON), MatrixJSON: datatypes.JSON([]byte("[]")), RiskItemsJSON: datatypes.JSON([]byte("[]")), HighestRiskLevel: constants.RiskLow, AlgorithmVersion: "v1", CreatedBy: 1}
	if err := e.runRepo.Create(e.ctx, &run, repository.AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run.ID
}

func (e *scoring006Env) status(t *testing.T, id uint) constants.AssessmentStatus {
	t.Helper()
	run, err := e.runRepo.Get(e.ctx, id)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	return run.AssessmentStatus
}

func snapshotFor(routeID, version uint) []byte {
	b, _ := json.Marshal(map[string]any{"route_id": routeID, "route_version_at_queue": version, "queued_at": time.Now().UTC()})
	return b
}

func TestRunVersionConflictResetsToQueued(t *testing.T) {
	e := newScoring006(t)
	route, err := e.routeRepo.Get(e.ctx, 1)
	if err != nil {
		t.Fatalf("get seed route: %v", err)
	}
	updated := model.ProcessRoute{ID: route.ID, ProductName: "changed", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
	if err := e.routeRepo.Update(e.ctx, &updated, route.Version, repository.AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("bump route version: %v", err)
	}
	id := e.queueRun(t, route.ID, snapshotFor(route.ID, route.Version))
	_, err = e.svc.Run(e.ctx, id, Principal{ID: 1, Role: "quality_analyst"}, "req")
	if err == nil {
		t.Fatal("expected version conflict error")
	}
	if got := e.status(t, id); got != constants.AssessmentQueued {
		t.Fatalf("run status after version conflict = %q, want queued (must roll back from calculating)", got)
	}
}

func TestRunComputeErrorResetsToQueued(t *testing.T) {
	e := newScoring006(t)
	// Build a route with a single step so graph decoding fails during compute.
	steps, _ := json.Marshal([]dto.RouteStep{{StepCode: "ONLY-01", StepName: "Only step", ProfileID: 1}})
	declared, _ := json.Marshal([]string{"Peanut"})
	route := model.ProcessRoute{RouteCode: "RT-ONE-006", ProductName: "Single step", OrderedStepsJSON: datatypes.JSON(steps), DeclaredAllergensJSON: datatypes.JSON(declared), RouteStatus: "active", Version: 1, OwnerID: 1}
	if err := e.routeRepo.Create(e.ctx, &route, repository.AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("create route: %v", err)
	}
	id := e.queueRun(t, route.ID, snapshotFor(route.ID, route.Version))
	_, err := e.svc.Run(e.ctx, id, Principal{ID: 1, Role: "quality_analyst"}, "req")
	if err == nil {
		t.Fatal("expected compute error")
	}
	if got := e.status(t, id); got != constants.AssessmentQueued {
		t.Fatalf("run status after compute error = %q, want queued", got)
	}
}

func TestRunSnapshotInvalidResetsToQueued(t *testing.T) {
	e := newScoring006(t)
	id := e.queueRun(t, 1, []byte("[]"))
	_, err := e.svc.Run(e.ctx, id, Principal{ID: 1, Role: "quality_analyst"}, "req")
	if err == nil {
		t.Fatal("expected snapshot error")
	}
	if got := e.status(t, id); got != constants.AssessmentQueued {
		t.Fatalf("run status after snapshot error = %q, want queued", got)
	}
}

func TestRunRouteGetErrorResetsToQueued(t *testing.T) {
	e := newScoring006(t)
	route, err := e.routeRepo.Get(e.ctx, 1)
	if err != nil {
		t.Fatalf("get seed route: %v", err)
	}
	id := e.queueRun(t, route.ID, snapshotFor(route.ID, route.Version))
	if err := e.db.DB.Where("id = ?", route.ID).Delete(&model.ProcessRoute{}).Error; err != nil {
		t.Fatalf("delete route: %v", err)
	}
	_, err = e.svc.Run(e.ctx, id, Principal{ID: 1, Role: "quality_analyst"}, "req")
	if err == nil {
		t.Fatal("expected route lookup error")
	}
	if got := e.status(t, id); got != constants.AssessmentQueued {
		t.Fatalf("run status after route lookup error = %q, want queued", got)
	}
}
