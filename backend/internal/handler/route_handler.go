package handler

import (
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"food-allergen-crosscontact-analyzer/backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type RouteHandler struct {
	service  *service.RouteService
	validate *validator.Validate
}

func NewRouteHandler(svc *service.RouteService, validate *validator.Validate) *RouteHandler {
	return &RouteHandler{service: svc, validate: validate}
}

func (h *RouteHandler) List(c *gin.Context) {
	var query dto.RouteQuery
	if !util.BindQuery(c, &query, h.validate) {
		return
	}
	items, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		util.Error(c, err)
		return
	}
	page, size := util.PageValues(query.Page, query.PageSize)
	util.Page(c, items, page, size, total)
}

func (h *RouteHandler) Get(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, item)
}

func (h *RouteHandler) Create(c *gin.Context) {
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.CreateRouteRequest
	if !util.BindJSON(c, &request, h.validate) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), request, principal, RequestID(c))
	if err != nil {
		util.Error(c, err)
		return
	}
	util.Created(c, item)
}

func (h *RouteHandler) Update(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.UpdateRouteRequest
	if !util.BindJSON(c, &request, h.validate) {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, request, principal, RequestID(c))
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, item)
}
