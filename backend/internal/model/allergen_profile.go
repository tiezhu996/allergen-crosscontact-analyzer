package model

import (
	"time"

	"gorm.io/datatypes"
)

type AllergenProfile struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	ProfileCode           string         `gorm:"size:64;uniqueIndex;not null" json:"profile_code"`
	MaterialName          string         `gorm:"size:180;not null" json:"material_name"`
	AllergensJSON         datatypes.JSON `gorm:"type:json;not null" json:"allergens_json"`
	SourceType            string         `gorm:"size:48;not null" json:"source_type"`
	SupplierStatementDate *time.Time     `gorm:"type:date" json:"supplier_statement_date"`
	ProfileStatus         string         `gorm:"size:24;not null;default:'active';check:profile_status IN ('draft','active','retired')" json:"profile_status"`
	Version               uint           `gorm:"not null;default:1" json:"version"`
	ReviewedBy            *uint          `gorm:"index" json:"reviewed_by"`
	CreatedBy             uint           `gorm:"index;not null" json:"created_by"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

func (AllergenProfile) TableName() string { return "allergen_profiles" }
