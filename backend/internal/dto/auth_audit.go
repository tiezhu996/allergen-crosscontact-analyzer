package dto

import "food-allergen-crosscontact-analyzer/backend/internal/constants"

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=64"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type UserView struct {
	ID          uint           `json:"id"`
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	Role        constants.Role `json:"role"`
}

type LoginResponse struct {
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expires_at"`
	User      UserView `json:"user"`
}

type AuditQuery struct {
	Page       int    `form:"page" validate:"omitempty,min=1"`
	PageSize   int    `form:"page_size" validate:"omitempty,min=1,max=100"`
	Action     string `form:"action" validate:"omitempty,max=80"`
	EntityType string `form:"entity_type" validate:"omitempty,max=64"`
	EntityID   uint   `form:"entity_id"`
	RequestID  string `form:"request_id" validate:"omitempty,max=64"`
}
