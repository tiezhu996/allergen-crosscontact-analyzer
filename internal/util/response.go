package util

import (
	"fmt"
	"net/http"
	"strconv"

	"food-allergen-crosscontact-analyzer/backend/internal/middleware"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type envelope struct {
	Success   bool       `json:"success"`
	Data      any        `json:"data,omitempty"`
	Meta      any        `json:"meta,omitempty"`
	Error     *errorBody `json:"error,omitempty"`
	RequestID string     `json:"request_id"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
type pageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data, RequestID: middleware.GetRequestID(c)})
}
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Success: true, Data: data, RequestID: middleware.GetRequestID(c)})
}
func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func Page(c *gin.Context, data any, page, size int, total int64) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	pages := totalPages(total, size)
	c.JSON(http.StatusOK, envelope{Success: true, Data: data, Meta: pageMeta{Page: page, PageSize: size, Total: total, TotalPages: pages}, RequestID: middleware.GetRequestID(c)})
}

func totalPages(total int64, size int) int {
	if size <= 0 || total <= 0 {
		return 0
	}
	pages := int(total / int64(size))
	if total%int64(size) != 0 {
		pages++
	}
	return pages
}

func Error(c *gin.Context, err error) {
	app := service.NormalizeError(err)
	_ = c.Error(err)
	c.JSON(app.Status, envelope{Success: false, Error: &errorBody{Code: app.Code, Message: app.Message, Details: app.Details}, RequestID: middleware.GetRequestID(c)})
}

func BindJSON(c *gin.Context, target any, validate *validator.Validate) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		Error(c, &service.AppError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "请求 JSON 无法解析", Cause: err})
		return false
	}
	if err := validate.Struct(target); err != nil {
		Error(c, validationError(err))
		return false
	}
	return true
}

func BindQuery(c *gin.Context, target any, validate *validator.Validate) bool {
	if err := c.ShouldBindQuery(target); err != nil {
		Error(c, &service.AppError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "查询参数无法解析", Cause: err})
		return false
	}
	if err := validate.Struct(target); err != nil {
		Error(c, validationError(err))
		return false
	}
	return true
}

func PathID(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		Error(c, &service.AppError{Status: http.StatusBadRequest, Code: "invalid_id", Message: fmt.Sprintf("路径参数 %s 必须为正整数", name), Cause: err})
		return 0, false
	}
	return uint(value), true
}

func Principal(c *gin.Context) (service.Principal, bool) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		Error(c, &service.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "需要有效登录凭证"})
		return service.Principal{}, false
	}
	return principal, true
}

func PageValues(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	return page, size
}

func validationError(err error) *service.AppError {
	details := make([]map[string]string, 0)
	if fields, ok := err.(validator.ValidationErrors); ok {
		for _, field := range fields {
			details = append(details, map[string]string{"field": field.Field(), "rule": field.Tag(), "value": fmt.Sprint(field.Value())})
		}
	}
	return &service.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "输入校验失败", Details: details, Cause: err}
}
