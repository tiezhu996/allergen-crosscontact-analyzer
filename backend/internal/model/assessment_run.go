package model

import (
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"gorm.io/datatypes"
)

type AssessmentRun struct {
	ID                uint                       `gorm:"primaryKey" json:"id"`
	RouteID           uint                       `gorm:"index;not null" json:"route_id"`
	AssessmentStatus  constants.AssessmentStatus `gorm:"size:32;index;not null;check:assessment_status IN ('queued','calculating','pending_review','accepted','rejected','stale')" json:"assessment_status"`
	InputSnapshotJSON datatypes.JSON             `gorm:"type:json;not null" json:"input_snapshot_json"`
	MatrixJSON        datatypes.JSON             `gorm:"type:json;not null" json:"matrix_json"`
	RiskItemsJSON     datatypes.JSON             `gorm:"type:json;not null" json:"risk_items_json"`
	HighestRiskLevel  constants.RiskLevel        `gorm:"size:16;not null;check:highest_risk_level IN ('low','medium','high','critical')" json:"highest_risk_level"`
	AlgorithmVersion  string                     `gorm:"size:64;not null" json:"algorithm_version"`
	CreatedBy         uint                       `gorm:"index;not null" json:"created_by"`
	ReviewedBy        *uint                      `gorm:"index" json:"reviewed_by"`
	ReviewReason      string                     `gorm:"type:text" json:"review_reason"`
	CompletedAt       *time.Time                 `json:"completed_at"`
	ReviewedAt        *time.Time                 `json:"reviewed_at"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

func (AssessmentRun) TableName() string { return "assessment_runs" }
