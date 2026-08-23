package service

import (
	"context"
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
)

func TestCreateRouteDoesNotMutateInputSteps(t *testing.T) {
	cfg := config.Config{
		DBDriver: "sqlite",
		DBDSN:    "file:scoring005s?mode=memory&cache=shared&_pragma=busy_timeout(10000)",
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
	profileRepo := repository.NewProfileRepository(db.DB)
	routeRepo := repository.NewRouteRepository(db.DB)
	svc := NewRouteService(routeRepo, profileRepo)
	req := dto.CreateRouteRequest{
		RouteCode:   "RT-CASE-005",
		ProductName: "Case sensitivity test",
		OrderedSteps: []dto.RouteStep{
			{StepCode: "mix-01", StepName: "Primary mixing", ProfileID: 1},
			{StepCode: "fill-02", StepName: "Shared filler", ProfileID: 2},
		},
		DeclaredAllergens: []string{"Milk"},
		RouteStatus:       "active",
	}
	if _, err := svc.Create(ctx, req, Principal{ID: 1, Role: "quality_analyst"}, "req"); err != nil {
		t.Fatalf("create route: %v", err)
	}
	if req.OrderedSteps[0].StepCode != "mix-01" {
		t.Fatalf("caller's input step code mutated to %q; must stay %q", req.OrderedSteps[0].StepCode, "mix-01")
	}
	if req.OrderedSteps[1].StepCode != "fill-02" {
		t.Fatalf("caller's input step code mutated to %q; must stay %q", req.OrderedSteps[1].StepCode, "fill-02")
	}
}
