package handler

import (
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"food-allergen-crosscontact-analyzer/backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AssessmentHandler struct {
	service  *service.AssessmentService
	validate *validator.Validate
}

func NewAssessmentHandler(svc *service.AssessmentService, validate *validator.Validate) *AssessmentHandler {
	return &AssessmentHandler{service: svc, validate: validate}
}

func (h *AssessmentHandler) Preview(c *gin.Context) {
	var request dto.MatrixRequest
	if !util.BindJSON(c, &request, h.validate) {
		return
	}
	result, err := h.service.Preview(c.Request.Context(), request.RouteID)
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, result)
}

func (h *AssessmentHandler) List(c *gin.Context) {
	var query dto.AssessmentQuery
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

func (h *AssessmentHandler) Summary(c *gin.Context) {
	result, err := h.service.Summary(c.Request.Context())
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, result)
}
func (h *AssessmentHandler) Get(c *gin.Context) {
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

func (h *AssessmentHandler) Create(c *gin.Context) {
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.CreateAssessmentRequest
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

func (h *AssessmentHandler) Run(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	item, err := h.service.Run(c.Request.Context(), id, principal, RequestID(c))
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, item)
}

func (h *AssessmentHandler) Review(c *gin.Context) {
	id, ok := util.PathID(c, "id")
	if !ok {
		return
	}
	principal, ok := util.Principal(c)
	if !ok {
		return
	}
	var request dto.ReviewAssessmentRequest
	if !util.BindJSON(c, &request, h.validate) {
		return
	}
	item, err := h.service.Review(c.Request.Context(), id, request, principal, RequestID(c))
	if err != nil {
		util.Error(c, err)
		return
	}
	util.OK(c, item)
}
