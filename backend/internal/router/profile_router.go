package router

import (
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerProfileRoutes(group *gin.RouterGroup, h *handler.ProfileHandler) {
	profiles := group.Group("/profiles")
	profiles.GET("", h.List)
	profiles.GET("/:id", h.Get)
	profiles.POST("", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Create)
	profiles.PUT("/:id", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Update)
}
