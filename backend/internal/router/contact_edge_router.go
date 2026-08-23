package router

import (
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerContactEdgeRoutes(group *gin.RouterGroup, h *handler.ContactEdgeHandler) {
	edges := group.Group("/contact-edges")
	edges.GET("", h.List)
	edges.GET("/:id", h.Get)
	edges.POST("", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Create)
	edges.PUT("/:id", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Update)
}
