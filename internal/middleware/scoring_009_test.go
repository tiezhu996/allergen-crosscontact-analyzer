package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestEngine(use func(...gin.HandlerFunc) *gin.Engine, middlewares ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.Use(middlewares...)
	engine.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func TestRateLimitNoNilMapPanic(t *testing.T) {
	engine := newTestEngine(nil, RateLimit(10))
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200 (rate limiter must not panic)", rec.Code)
	}
}

func TestRateLimitSecondRequestNotDeadlocked(t *testing.T) {
	engine := newTestEngine(nil, RateLimit(10))
	done := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
			done <- rec.Code
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case code := <-done:
			if code != http.StatusOK {
				t.Fatalf("request status = %d, want 200", code)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("request %d deadlocked: the limiter lock was never released", i+1)
		}
	}
}

func TestRecoveryNoDoubleWrite(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID(), Recovery(logger))
	engine.GET("/panic", func(c *gin.Context) {
		c.String(http.StatusOK, "partial")
		panic("boom")
	})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if rec.Body.String() != "partial" {
		t.Fatalf("response body = %q, want exactly %q (recovery must not append a second body)", rec.Body.String(), "partial")
	}
}

func TestCORSDisallowedOriginNoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CORS([]string{"http://allowed.example"}))
	engine.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("CORS header %q returned for a disallowed origin", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

var _ = strings.Builder{}
