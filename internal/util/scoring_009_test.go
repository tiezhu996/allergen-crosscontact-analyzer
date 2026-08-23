package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
)

func TestErrorUsesAppStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("request_id", "req-1"); c.Next() })
	engine.GET("/x", func(c *gin.Context) {
		Error(c, &service.AppError{Status: http.StatusNotFound, Code: "not_found", Message: "nope"})
	})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d, want 404 (AppError status must be honored)", rec.Code)
	}
}

func TestPageTotalPagesCorrect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("request_id", "req-2"); c.Next() })
	engine.GET("/list", func(c *gin.Context) { Page(c, []int{1, 2, 3}, 1, 20, 25) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/list", nil))
	var body struct {
		Meta struct {
			Total      int64 `json:"total"`
			TotalPages int   `json:"total_pages"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Meta.Total != 25 || body.Meta.TotalPages != 2 {
		t.Fatalf("meta = total %d pages %d, want total 25 pages 2", body.Meta.Total, body.Meta.TotalPages)
	}
}

func TestPageHandlesZeroSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("request_id", "req-3"); c.Next() })
	engine.GET("/list", func(c *gin.Context) { Page(c, []int{1}, 1, 0, 3) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/list", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("page with size 0 = %d, want 200 (zero size must be defaulted, not crash)", rec.Code)
	}
}
