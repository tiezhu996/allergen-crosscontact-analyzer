package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestContactEdgeRequestRejectsOutOfRangeFactor(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	req := CreateContactEdgeRequest{
		RouteID: 1, FromStepCode: "A-01", ToStepCode: "B-02", ContactType: "sequence",
		SharedEquipment: "eq", CleaningFactor: 1.5, CarryoverProbability: 0.5,
		EvidenceNote: "out of range", Enabled: true,
	}
	if err := validate.Struct(req); err == nil {
		t.Fatal("cleaning factor 1.5 accepted; range validation must reject it")
	}
}

func TestCreateProfileRejectsInvalidStatus(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	req := CreateProfileRequest{
		ProfileCode: "MAT-BAD-008", MaterialName: "Bad paste", Allergens: []string{"Peanut"},
		SourceType: "supplier_statement", ProfileStatus: "bogus",
	}
	if err := validate.Struct(req); err == nil {
		t.Fatal("profile status 'bogus' accepted; oneof validation must reject it")
	}
}

func TestCreateRouteRejectsInvalidStatus(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	req := CreateRouteRequest{
		RouteCode: "RT-BAD-008", ProductName: "Bad route",
		OrderedSteps: []RouteStep{{StepCode: "A-01", StepName: "Step A", ProfileID: 1}, {StepCode: "B-02", StepName: "Step B", ProfileID: 2}},
		DeclaredAllergens: []string{"Milk"}, RouteStatus: "bogus",
	}
	if err := validate.Struct(req); err == nil {
		t.Fatal("route status 'bogus' accepted; oneof validation must reject it")
	}
}
