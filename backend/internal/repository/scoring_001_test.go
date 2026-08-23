package repository

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/datatypes"
)

func newScoringDB(t *testing.T) (*Database, context.Context) {
	t.Helper()
	cfg := config.Config{
		DBDriver: "sqlite",
		DBDSN:    "file:scoring001?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
	}
	db, err := Open(cfg)
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
	return db, ctx
}

func seedRouteAndRun(t *testing.T, db *Database, ctx context.Context) (route model.ProcessRoute, run model.AssessmentRun) {
	t.Helper()
	routeRepo := NewRouteRepository(db.DB)
	runRepo := NewAssessmentRepository(db.DB)
	steps, _ := json.Marshal([]dto.RouteStep{
		{StepCode: "A-01", StepName: "Step A", ProfileID: 1},
		{StepCode: "B-02", StepName: "Step B", ProfileID: 1},
	})
	declared, _ := json.Marshal([]string{"Peanut"})
	route = model.ProcessRoute{RouteCode: "RT-TEST-001", ProductName: "Test product", OrderedStepsJSON: datatypes.JSON(steps), DeclaredAllergensJSON: datatypes.JSON(declared), RouteStatus: "active", Version: 1, OwnerID: 1}
	if err := routeRepo.Create(ctx, &route, AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("create route: %v", err)
	}
	snapshot, _ := json.Marshal(map[string]any{"route_id": route.ID, "route_version_at_queue": route.Version, "queued_at": time.Now().UTC()})
	run = model.AssessmentRun{RouteID: route.ID, AssessmentStatus: constants.AssessmentQueued, InputSnapshotJSON: datatypes.JSON(snapshot), MatrixJSON: datatypes.JSON([]byte("[]")), RiskItemsJSON: datatypes.JSON([]byte("[]")), HighestRiskLevel: constants.RiskLow, AlgorithmVersion: "v1", CreatedBy: 1}
	if err := runRepo.Create(ctx, &run, AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return route, run
}

// The route update lands while the assessment is still calculating; the in-flight
// run must end stale (not pending_review) and must not be overwritten afterwards.
func TestRouteUpdateDuringCalcStalesRun(t *testing.T) {
	db, ctx := newScoringDB(t)
	route, run := seedRouteAndRun(t, db, ctx)
	routeRepo := NewRouteRepository(db.DB)
	runRepo := NewAssessmentRepository(db.DB)
	scope := AuditContext{RequestID: "race-req", ActorID: 1, ActorName: "analyst"}

	calcStarted := make(chan struct{})
	updateDone := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := runRepo.BeginCalculation(ctx, run.ID, scope); err != nil {
			t.Errorf("begin calculation: %v", err)
			close(calcStarted)
			return
		}
		close(calcStarted)
		<-updateDone
		_ = runRepo.CompleteCalculation(ctx, run.ID, run.InputSnapshotJSON, datatypes.JSON([]byte("[]")), datatypes.JSON([]byte("[]")), constants.RiskLow, "v1", scope)
	}()
	go func() {
		defer wg.Done()
		<-calcStarted
		updated := model.ProcessRoute{ID: route.ID, ProductName: "Test product v2", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
		if err := routeRepo.Update(ctx, &updated, route.Version, scope); err != nil {
			t.Errorf("route update: %v", err)
		}
		close(updateDone)
	}()
	wg.Wait()

	after, err := runRepo.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if after.AssessmentStatus != constants.AssessmentStale {
		t.Fatalf("run status = %q, want stale (in-flight run must be invalidated by route change)", after.AssessmentStatus)
	}
}

// Profile updates invalidate every route's in-flight runs as well.
func TestProfileUpdateDuringCalcStalesRun(t *testing.T) {
	db, ctx := newScoringDB(t)
	_, run := seedRouteAndRun(t, db, ctx)
	profileRepo := NewProfileRepository(db.DB)
	runRepo := NewAssessmentRepository(db.DB)
	scope := AuditContext{RequestID: "race-req2", ActorID: 1, ActorName: "analyst"}

	calcStarted := make(chan struct{})
	updateDone := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := runRepo.BeginCalculation(ctx, run.ID, scope); err != nil {
			t.Errorf("begin calculation: %v", err)
			close(calcStarted)
			return
		}
		close(calcStarted)
		<-updateDone
		_ = runRepo.CompleteCalculation(ctx, run.ID, run.InputSnapshotJSON, datatypes.JSON([]byte("[]")), datatypes.JSON([]byte("[]")), constants.RiskLow, "v1", scope)
	}()
	go func() {
		defer wg.Done()
		<-calcStarted
		allergens, _ := json.Marshal([]string{"Peanut", "Soy"})
		date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		profile := model.AllergenProfile{ID: 1, MaterialName: "Roasted peanut paste v2", AllergensJSON: datatypes.JSON(allergens), SourceType: "supplier_statement", SupplierStatementDate: &date, ProfileStatus: "active"}
		if err := profileRepo.Update(ctx, &profile, 1, scope); err != nil {
			t.Errorf("profile update: %v", err)
		}
		close(updateDone)
	}()
	wg.Wait()

	after, err := runRepo.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if after.AssessmentStatus != constants.AssessmentStale {
		t.Fatalf("run status = %q, want stale after profile change", after.AssessmentStatus)
	}
}

// Staling an in-flight run through a route change must leave an audit trail.
func TestRouteUpdateStaleAuditRecorded(t *testing.T) {
	db, ctx := newScoringDB(t)
	route, run := seedRouteAndRun(t, db, ctx)
	routeRepo := NewRouteRepository(db.DB)
	runRepo := NewAssessmentRepository(db.DB)
	scope := AuditContext{RequestID: "audit-req", ActorID: 1, ActorName: "analyst"}
	if err := runRepo.BeginCalculation(ctx, run.ID, scope); err != nil {
		t.Fatalf("begin calculation: %v", err)
	}
	var beforeAudits int64
	if err := db.DB.Model(&model.AuditEvent{}).Where("entity_type = ? AND entity_id = ?", "assessment_run", run.ID).Count(&beforeAudits).Error; err != nil {
		t.Fatalf("count audits before: %v", err)
	}
	updated := model.ProcessRoute{ID: route.ID, ProductName: "renamed", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
	if err := routeRepo.Update(ctx, &updated, route.Version, scope); err != nil {
		t.Fatalf("route update: %v", err)
	}
	after, err := runRepo.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if after.AssessmentStatus != constants.AssessmentStale {
		t.Fatalf("run status = %q, want stale", after.AssessmentStatus)
	}
	var afterAudits int64
	if err := db.DB.Model(&model.AuditEvent{}).Where("entity_type = ? AND entity_id = ?", "assessment_run", run.ID).Count(&afterAudits).Error; err != nil {
		t.Fatalf("count audits after: %v", err)
	}
	if afterAudits <= beforeAudits {
		t.Fatalf("staling the run recorded no audit event (%d -> %d)", beforeAudits, afterAudits)
	}
}

// Updating a route with a stale expected version must be rejected.
func TestStaleVersionRouteUpdateRejected(t *testing.T) {
	db, ctx := newScoringDB(t)
	route, _ := seedRouteAndRun(t, db, ctx)
	routeRepo := NewRouteRepository(db.DB)
	scope := AuditContext{RequestID: "ver-req", ActorID: 1, ActorName: "analyst"}
	updated := model.ProcessRoute{ID: route.ID, ProductName: "renamed", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
	if err := routeRepo.Update(ctx, &updated, route.Version+99, scope); err == nil {
		t.Fatal("stale-version route update unexpectedly succeeded; optimistic lock must reject it")
	}
}

// A successful route update must advance the version so the next stale write is blocked.
func TestRouteUpdateAdvancesVersion(t *testing.T) {
	db, ctx := newScoringDB(t)
	route, _ := seedRouteAndRun(t, db, ctx)
	routeRepo := NewRouteRepository(db.DB)
	scope := AuditContext{RequestID: "adv-req", ActorID: 1, ActorName: "analyst"}
	updated := model.ProcessRoute{ID: route.ID, ProductName: "renamed", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
	if err := routeRepo.Update(ctx, &updated, route.Version, scope); err != nil {
		t.Fatalf("first update: %v", err)
	}
	after, err := routeRepo.Get(ctx, route.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}
	if after.Version != route.Version+1 {
		t.Fatalf("version = %d, want %d (must advance after update)", after.Version, route.Version+1)
	}
	second := model.ProcessRoute{ID: route.ID, ProductName: "renamed again", OrderedStepsJSON: route.OrderedStepsJSON, DeclaredAllergensJSON: route.DeclaredAllergensJSON, RouteStatus: "active"}
	if err := routeRepo.Update(ctx, &second, route.Version, scope); err == nil {
		t.Fatal("second update with the original version unexpectedly succeeded")
	}
}
