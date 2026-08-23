package model

import "time"

type ContactEdge struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	RouteID              uint      `gorm:"index:idx_route_edge,priority:1;not null" json:"route_id"`
	FromStepCode         string    `gorm:"size:64;index:idx_route_edge,priority:2;not null" json:"from_step_code"`
	ToStepCode           string    `gorm:"size:64;index:idx_route_edge,priority:3;not null" json:"to_step_code"`
	ContactType          string    `gorm:"size:48;not null" json:"contact_type"`
	SharedEquipment      string    `gorm:"size:180;not null" json:"shared_equipment"`
	CleaningFactor       float64   `gorm:"not null" json:"cleaning_factor"`
	CarryoverProbability float64   `gorm:"not null" json:"carryover_probability"`
	EvidenceNote         string    `gorm:"type:text;not null" json:"evidence_note"`
	Enabled              bool      `gorm:"not null;default:true" json:"enabled"`
	Version              uint      `gorm:"not null;default:1" json:"version"`
	CreatedBy            uint      `gorm:"index;not null" json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (ContactEdge) TableName() string { return "contact_edges" }
