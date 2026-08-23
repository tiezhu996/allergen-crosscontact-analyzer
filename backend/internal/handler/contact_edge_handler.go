package handler

import (
	"fmt"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"food-allergen-crosscontact-analyzer/backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ContactEdgeHandler struct {
	service  *service.ContactEdgeService
	validate *validator.Validate
}

func NewContactEdgeHandler(svc *service.ContactEdgeService, validate *validator.Validate) *ContactEdgeHandler {
	return &ContactEdgeHandler{service: svc, validate: validate}
}

func (h *ContactEdgeHandler) List(c *gin.Context) {
	var query dto.ContactEdgeQuery
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

func (h *ContactEdgeHandler) Get(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		util.Error(c, fmt.Errorf("get contact edge: %v", err))
		return
	}
	util.OK(c, item)
}

func (h *ContactEdgeHandler) Create(c *gin.Context) {
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.CreateContactEdgeRequest
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

func (h *ContactEdgeHandler) Update(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.UpdateContactEdgeRequest
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
