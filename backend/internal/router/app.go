package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/handler"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Support     *handler.SupportHandler
	Profiles    *handler.ProfileHandler
	Routes      *handler.RouteHandler
	Edges       *handler.ContactEdgeHandler
	Assessments *handler.AssessmentHandler
}

func New(cfg config.Config, logger *slog.Logger, database *repository.Database, support *service.SupportService, handlers Handlers) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.AccessLog(logger), middleware.CORS(cfg.CORSOrigins), middleware.RateLimit(cfg.RateLimitPerMinute))
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "food-allergen-crosscontact-analyzer", "time": time.Now().UTC()})
	})
	engine.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := database.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "available"})
	})
	api := engine.Group("/api/v1")
	api.POST("/auth/login", handlers.Support.Login)
	protected := api.Group("")
	protected.Use(middleware.AuthWithResolver(support.ParseToken, support.CurrentPrincipal))
	protected.GET("/auth/me", handlers.Support.Me)
	registerProfileRoutes(protected, handlers.Profiles)
	registerRouteRoutes(protected, handlers.Routes)
	registerContactEdgeRoutes(protected, handlers.Edges)
	registerAssessmentRoutes(protected, handlers.Assessments)
	protected.GET("/audit", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin), handlers.Support.Audit)
	protected.GET("/versions/:entityType/:id", handlers.Support.VersionDiff)
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "route_not_found", "message": "接口不存在"}, "request_id": middleware.GetRequestID(c)})
	})
	engine.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "error": gin.H{"code": "method_not_allowed", "message": "请求方法不受支持"}, "request_id": middleware.GetRequestID(c)})
	})
	return engine
}

func timeboxedContext(c *gin.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), duration)
}
