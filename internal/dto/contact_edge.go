package dto

type CreateContactEdgeRequest struct {
	RouteID              uint    `json:"route_id" validate:"required,min=1"`
	FromStepCode         string  `json:"from_step_code" validate:"required,min=2,max=64,nefield=ToStepCode"`
	ToStepCode           string  `json:"to_step_code" validate:"required,min=2,max=64"`
	ContactType          string  `json:"contact_type" validate:"required,oneof=sequence shared_line rework airborne manual_transfer"`
	SharedEquipment      string  `json:"shared_equipment" validate:"required,min=2,max=180"`
	CleaningFactor       float64 `json:"cleaning_factor" validate:"gte=0,lte=1"`
	CarryoverProbability float64 `json:"carryover_probability" validate:"gte=0,lte=1"`
	EvidenceNote         string  `json:"evidence_note" validate:"required,min=4,max=1000"`
	Enabled              bool    `json:"enabled"`
}

type UpdateContactEdgeRequest struct {
	ContactType          string  `json:"contact_type" validate:"required,oneof=sequence shared_line rework airborne manual_transfer"`
	SharedEquipment      string  `json:"shared_equipment" validate:"required,min=2,max=180"`
	CleaningFactor       float64 `json:"cleaning_factor" validate:"gte=0,lte=1"`
	CarryoverProbability float64 `json:"carryover_probability" validate:"gte=0,lte=1"`
	EvidenceNote         string  `json:"evidence_note" validate:"required,min=4,max=1000"`
	Enabled              bool    `json:"enabled"`
	ExpectedVersion      uint    `json:"expected_version" validate:"required,min=1"`
}

type ContactEdgeQuery struct {
	Page     int    `form:"page" validate:"omitempty,min=1"`
	PageSize int    `form:"page_size" validate:"omitempty,min=1,max=100"`
	RouteID  uint   `form:"route_id"`
	Enabled  string `form:"enabled" validate:"omitempty,oneof=true false"`
}
