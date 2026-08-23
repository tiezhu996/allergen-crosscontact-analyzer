package router

import (
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerAssessmentRoutes(group *gin.RouterGroup, h *handler.AssessmentHandler) {
	group.POST("/matrix/compute", h.Preview)
	runs := group.Group("/assessments")
	runs.GET("", h.List)
	runs.GET("/summary", h.Summary)
	runs.GET("/:id", h.Get)
	runs.POST("", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Create)
	runs.POST("/:id/run", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Run)
	runs.POST("/:id/review", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin), h.Review)
}
