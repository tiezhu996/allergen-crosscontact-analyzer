package dto

type MatrixRequest struct {
	RouteID uint `json:"route_id" validate:"required,min=1"`
}

type CreateAssessmentRequest struct {
	RouteID uint `json:"route_id" validate:"required,min=1"`
}

type ReviewAssessmentRequest struct {
	Decision string `json:"decision" validate:"required,oneof=accepted rejected"`
	Reason   string `json:"reason" validate:"required,min=4,max=1000"`
}

type AssessmentQuery struct {
	Page     int    `form:"page" validate:"omitempty,min=1"`
	PageSize int    `form:"page_size" validate:"omitempty,min=1,max=100"`
	RouteID  uint   `form:"route_id"`
	Status   string `form:"status" validate:"omitempty,oneof=queued calculating pending_review accepted rejected stale"`
}

type AssessmentSummary struct {
	Total         int64            `json:"total"`
	ByStatus      map[string]int64 `json:"by_status"`
	PendingReview int64            `json:"pending_review"`
}
