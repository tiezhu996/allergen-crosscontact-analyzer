package repository

import (
	"context"
	"encoding/json"
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/datatypes"
)

func newScoring003DB(t *testing.T) (*Database, context.Context) {
	t.Helper()
	cfg := config.Config{
		DBDriver: "sqlite",
		DBDSN:    "file:scoring003?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
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

func createRun(t *testing.T, db *Database, ctx context.Context, status constants.AssessmentStatus) model.AssessmentRun {
	t.Helper()
	runRepo := NewAssessmentRepository(db.DB)
	snapshot, _ := json.Marshal(map[string]any{"route_id": 1, "route_version_at_queue": 1})
	run := model.AssessmentRun{RouteID: 1, AssessmentStatus: status, InputSnapshotJSON: datatypes.JSON(snapshot), MatrixJSON: datatypes.JSON([]byte("[]")), RiskItemsJSON: datatypes.JSON([]byte("[]")), HighestRiskLevel: constants.RiskLow, AlgorithmVersion: "v1", CreatedBy: 1}
	if err := runRepo.Create(ctx, &run, AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if status != constants.AssessmentQueued {
		if err := db.DB.Model(&model.AssessmentRun{}).Where("id = ?", run.ID).Update("assessment_status", status).Error; err != nil {
			t.Fatalf("set status: %v", err)
		}
	}
	return run
}

// Summary must tolerate existing runs instead of panicking on a nil map.
func TestAssessmentSummaryNoPanic(t *testing.T) {
	db, ctx := newScoring003DB(t)
	createRun(t, db, ctx, constants.AssessmentQueued)
	createRun(t, db, ctx, constants.AssessmentPendingReview)
	summary, err := NewAssessmentRepository(db.DB).Summary(ctx)
	if err != nil {
		t.Fatalf("summary errored: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("summary total = %d, want 2", summary.Total)
	}
}

// Pending review count must be reported.
func TestAssessmentSummaryPendingReviewCount(t *testing.T) {
	db, ctx := newScoring003DB(t)
	createRun(t, db, ctx, constants.AssessmentPendingReview)
	summary, err := NewAssessmentRepository(db.DB).Summary(ctx)
	if err != nil {
		t.Fatalf("summary errored: %v", err)
	}
	if summary.PendingReview != 1 {
		t.Fatalf("pending_review count = %d, want 1", summary.PendingReview)
	}
}

// A profile that is referenced by no route must yield a non-nil empty usage list.
func TestProfileDetailEmptyUsageNotNull(t *testing.T) {
	db, ctx := newScoring003DB(t)
	profileRepo := NewProfileRepository(db.DB)
	scope := AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}
	allergens, _ := json.Marshal([]string{"Tree Nut"})
	profile := model.AllergenProfile{ProfileCode: "MAT-NUT-UNUSED", MaterialName: "Unused nut paste", AllergensJSON: datatypes.JSON(allergens), SourceType: "supplier_statement", ProfileStatus: "active", Version: 1, CreatedBy: 1}
	if err := profileRepo.Create(ctx, &profile, scope); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	usage, err := profileRepo.Usage(ctx, profile.ID)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if usage == nil {
		t.Fatal("usage must be a non-nil empty slice so the detail JSON renders [] instead of null")
	}
	if len(usage) != 0 {
		t.Fatalf("usage length = %d, want 0", len(usage))
	}
}

// A route using the same profile in multiple steps must produce one usage entry.
func TestProfileUsageNoDuplicatePerRoute(t *testing.T) {
	db, ctx := newScoring003DB(t)
	profileRepo := NewProfileRepository(db.DB)
	routeRepo := NewRouteRepository(db.DB)
	scope := AuditContext{RequestID: "req", ActorID: 1, ActorName: "analyst"}
	allergens, _ := json.Marshal([]string{"Sesame"})
	profile := model.AllergenProfile{ProfileCode: "MAT-SESAME-003", MaterialName: "Sesame paste", AllergensJSON: datatypes.JSON(allergens), SourceType: "supplier_statement", ProfileStatus: "active", Version: 1, CreatedBy: 1}
	if err := profileRepo.Create(ctx, &profile, scope); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	steps, _ := json.Marshal([]dto.RouteStep{
		{StepCode: "A-01", StepName: "Step A", ProfileID: profile.ID},
		{StepCode: "B-02", StepName: "Step B", ProfileID: profile.ID},
	})
	declared, _ := json.Marshal([]string{"Sesame"})
	route := model.ProcessRoute{RouteCode: "RT-USAGE-003", ProductName: "Usage test", OrderedStepsJSON: datatypes.JSON(steps), DeclaredAllergensJSON: datatypes.JSON(declared), RouteStatus: "active", Version: 1, OwnerID: 1}
	if err := routeRepo.Create(ctx, &route, scope); err != nil {
		t.Fatalf("create route: %v", err)
	}
	usage, err := profileRepo.Usage(ctx, profile.ID)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if len(usage) != 1 {
		t.Fatalf("usage length = %d, want 1 (one entry per route even when profile appears twice)", len(usage))
	}
	if usage[0].RouteID != route.ID {
		t.Fatalf("usage route = %d, want %d", usage[0].RouteID, route.ID)
	}
}

