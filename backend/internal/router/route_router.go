package router

import (
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerRouteRoutes(group *gin.RouterGroup, h *handler.RouteHandler) {
	routes := group.Group("/routes")
	routes.GET("", h.List)
	routes.GET("/:id", h.Get)
	routes.POST("", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Create)
	routes.PUT("/:id", middleware.RBAC(constants.RoleQualityAnalyst, constants.RoleAdmin), h.Update)
}
