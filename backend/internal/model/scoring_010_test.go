package model

import (
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openScoring010(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:scoring010?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAssessmentStalePersistAllowed(t *testing.T) {
	db := openScoring010(t, &AssessmentRun{})
	run := AssessmentRun{RouteID: 1, AssessmentStatus: constants.AssessmentStale, InputSnapshotJSON: []byte("{}"), MatrixJSON: []byte("[]"), RiskItemsJSON: []byte("[]"), HighestRiskLevel: constants.RiskLow, AlgorithmVersion: "v1", CreatedBy: 1}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("persisting a stale assessment failed: %v (stale must be a legal status)", err)
	}
}

func TestProfileInvalidStatusRejected(t *testing.T) {
	db := openScoring010(t, &AllergenProfile{})
	profile := AllergenProfile{ProfileCode: "MAT-BOGUS", MaterialName: "Bogus", AllergensJSON: []byte(`["Peanut"]`), SourceType: "supplier_statement", ProfileStatus: "bogus", Version: 1, CreatedBy: 1}
	if err := db.Create(&profile).Error; err == nil {
		t.Fatal("profile with an illegal status was persisted; the status check must reject it")
	}
}

func TestEdgeOutOfRangeRejected(t *testing.T) {
	db := openScoring010(t, &ContactEdge{})
	edge := ContactEdge{RouteID: 1, FromStepCode: "A-01", ToStepCode: "B-02", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 1.5, CarryoverProbability: 0.5, EvidenceNote: "note", Enabled: true, Version: 1, CreatedBy: 1}
	if err := db.Create(&edge).Error; err == nil {
		t.Fatal("edge with cleaning factor > 1 was persisted; the range check must reject it")
	}
}

func TestRouteInvalidStatusRejected(t *testing.T) {
	db := openScoring010(t, &ProcessRoute{})
	route := ProcessRoute{RouteCode: "RT-BOGUS", ProductName: "Bogus route", OrderedStepsJSON: []byte("[]"), DeclaredAllergensJSON: []byte("[]"), RouteStatus: "bogus", Version: 1, OwnerID: 1}
	if err := db.Create(&route).Error; err == nil {
		t.Fatal("route with an illegal status was persisted; the status check must reject it")
	}
}

func TestAssessmentInvalidRiskRejected(t *testing.T) {
	db := openScoring010(t, &AssessmentRun{})
	run := AssessmentRun{RouteID: 1, AssessmentStatus: constants.AssessmentQueued, InputSnapshotJSON: []byte("{}"), MatrixJSON: []byte("[]"), RiskItemsJSON: []byte("[]"), HighestRiskLevel: "bogus", AlgorithmVersion: "v1", CreatedBy: 1}
	if err := db.Create(&run).Error; err == nil {
		t.Fatal("assessment with an illegal risk level was persisted; the check must reject it")
	}
}

func TestUserInvalidRoleRejected(t *testing.T) {
	db := openScoring010(t, &User{})
	user := User{Username: "bogus_user", PasswordHash: "hash", DisplayName: "Bogus", Role: "superuser", Active: true}
	if err := db.Create(&user).Error; err == nil {
		t.Fatal("user with an illegal role was persisted; the check must reject it")
	}
}
