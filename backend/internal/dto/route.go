package dto

type RouteStep struct {
	StepCode  string `json:"step_code" validate:"required,min=2,max=64"`
	StepName  string `json:"step_name" validate:"required,min=2,max=120"`
	ProfileID uint   `json:"profile_id" validate:"required,min=1"`
}

type CreateRouteRequest struct {
	RouteCode         string      `json:"route_code" validate:"required,min=2,max=64"`
	ProductName       string      `json:"product_name" validate:"required,min=2,max=180"`
	OrderedSteps      []RouteStep `json:"ordered_steps" validate:"required,min=2,max=40,dive"`
	DeclaredAllergens []string    `json:"declared_allergens" validate:"max=30,dive,required,max=80"`
	RouteStatus       string      `json:"route_status" validate:"required,oneof=draft active retired"`
}

type UpdateRouteRequest struct {
	ProductName       string      `json:"product_name" validate:"required,min=2,max=180"`
	OrderedSteps      []RouteStep `json:"ordered_steps" validate:"required,min=2,max=40,dive"`
	DeclaredAllergens []string    `json:"declared_allergens" validate:"max=30,dive,required,max=80"`
	RouteStatus       string      `json:"route_status" validate:"required,oneof=draft active retired"`
	ExpectedVersion   uint        `json:"expected_version" validate:"required,min=1"`
}

type RouteQuery struct {
	Page     int    `form:"page" validate:"omitempty,min=1"`
	PageSize int    `form:"page_size" validate:"omitempty,min=1,max=100"`
	Search   string `form:"search" validate:"omitempty,max=100"`
	Status   string `form:"status" validate:"omitempty,oneof=draft active retired"`
}

type VersionDiff struct {
	EntityType  string         `json:"entity_type"`
	EntityID    uint           `json:"entity_id"`
	Version     uint           `json:"version"`
	LatestAudit map[string]any `json:"latest_audit"`
}
