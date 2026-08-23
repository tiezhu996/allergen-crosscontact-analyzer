package model

import (
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"gorm.io/datatypes"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	DisplayName  string         `gorm:"size:120;not null" json:"display_name"`
	Role         constants.Role `gorm:"size:32;not null" json:"role"`
	Active       bool           `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type AuditEvent struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	RequestID     string         `gorm:"size:64;index;not null" json:"request_id"`
	ActorID       uint           `gorm:"index;not null" json:"actor_id"`
	ActorName     string         `gorm:"size:120;not null" json:"actor_name"`
	Action        string         `gorm:"size:80;index;not null" json:"action"`
	EntityType    string         `gorm:"size:64;index;not null" json:"entity_type"`
	EntityID      uint           `gorm:"index;not null" json:"entity_id"`
	BeforeSummary string         `gorm:"type:text" json:"before_summary"`
	AfterSummary  string         `gorm:"type:text" json:"after_summary"`
	MetadataJSON  datatypes.JSON `gorm:"type:json;not null" json:"metadata_json"`
	CreatedAt     time.Time      `gorm:"index;not null" json:"created_at"`
}

func (AuditEvent) TableName() string { return "audit_events" }
