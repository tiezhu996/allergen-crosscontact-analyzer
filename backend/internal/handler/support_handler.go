package handler

import (
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"food-allergen-crosscontact-analyzer/backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type SupportHandler struct {
	service  *service.SupportService
	validate *validator.Validate
}

func NewSupportHandler(svc *service.SupportService, validate *validator.Validate) *SupportHandler {
	return &SupportHandler{service: svc, validate: validate}
}

func (h *SupportHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if !util.BindJSON(c, &request, h.validate) {
		return
	}
	response, err := h.service.Login(c.Request.Context(), request)
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, response)
}

func (h *SupportHandler) Me(c *gin.Context) {
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	user, err := h.service.CurrentUser(c.Request.Context(), principal)
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, user)
}

func (h *SupportHandler) Audit(c *gin.Context) {
	var query dto.AuditQuery
	if !util.BindQuery(c, &query, h.validate) {
		return
	}
	events, total, err := h.service.ListAudit(c.Request.Context(), query)
	if err != nil {
		util.Error(c, err)
		return
	}
	page, size := util.PageValues(query.Page, query.PageSize)
	util.Page(c, events, page, size, total)
}

func (h *SupportHandler) VersionDiff(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	var versionQuery struct {
		Version uint `form:"version" validate:"required,min=1"`
	}
	if !util.BindQuery(c, &versionQuery, h.validate) {
		return
	}
	diff, err := h.service.VersionDiff(c.Request.Context(), c.Param("entityType"), id, versionQuery.Version)
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, diff)
}

func RequestID(c *gin.Context) string { return middleware.GetRequestID(c) }
