package model

import (
	"time"

	"gorm.io/datatypes"
)

type ProcessRoute struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	RouteCode             string         `gorm:"size:64;uniqueIndex;not null" json:"route_code"`
	ProductName           string         `gorm:"size:180;not null" json:"product_name"`
	OrderedStepsJSON      datatypes.JSON `gorm:"type:json;not null" json:"ordered_steps_json"`
	DeclaredAllergensJSON datatypes.JSON `gorm:"type:json;not null" json:"declared_allergens_json"`
	RouteStatus           string         `gorm:"size:24;not null;default:'draft'" json:"route_status"`
	Version               uint           `gorm:"not null;default:1" json:"version"`
	OwnerID               uint           `gorm:"index;not null" json:"owner_id"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

func (ProcessRoute) TableName() string { return "process_routes" }
