package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	RequestIDKey = "request_id"
	PrincipalKey = "principal"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		principal, _ := GetPrincipal(c)
		logger.Info("http_request", "request_id", GetRequestID(c), "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(), "bytes", c.Writer.Size(), "duration_ms", time.Since(started).Milliseconds(), "client_ip", c.ClientIP(), "actor_id", principal.ID, "errors", len(c.Errors))
	}
}

func Auth(parse func(string) (service.Principal, error)) gin.HandlerFunc {
	return AuthWithResolver(parse, nil)
}

// AuthWithResolver verifies the token and optionally resolves the principal
// from current account state before downstream RBAC checks run.
func AuthWithResolver(parse func(string) (service.Principal, error), resolve func(context.Context, service.Principal) (service.Principal, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abort(c, http.StatusUnauthorized, "unauthorized", "需要有效登录凭证")
			return
		}
		principal, err := parse(strings.TrimSpace(parts[1]))
		if err != nil {
			app := service.NormalizeError(err)
			abort(c, app.Status, app.Code, app.Message)
			return
		}
		if resolve != nil {
			principal, err = resolve(c.Request.Context(), principal)
			if err != nil {
				app := service.NormalizeError(err)
				abort(c, app.Status, app.Code, app.Message)
				return
			}
		}
		c.Set(PrincipalKey, principal)
		c.Next()
	}
}

func RBAC(roles ...constants.Role) gin.HandlerFunc {
	allowed := make(map[constants.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			abort(c, http.StatusUnauthorized, "unauthorized", "需要有效登录凭证")
			return
		}
		if !allowed[principal.Role] {
			abort(c, http.StatusForbidden, "forbidden", "当前角色无权执行此操作")
			return
		}
		c.Next()
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic_recovered", "request_id", GetRequestID(c), "panic", recovered, "stack", string(debug.Stack()))
				if c.Writer.Written() {
					abort(c, http.StatusInternalServerError, "internal_error", "服务处理失败")
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}

type rateBucket struct {
	window time.Time
	count  int
}
type limiter struct {
	mu          sync.Mutex
	limit       int
	buckets     map[string]rateBucket
	lastCleanup time.Time
}

func RateLimit(limitPerMinute int) gin.HandlerFunc {
	state := &limiter{limit: limitPerMinute, lastCleanup: time.Now()}
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		allowed, remaining := state.allow(key, now)
		c.Header("X-RateLimit-Limit", itoa(state.limit))
		c.Header("X-RateLimit-Remaining", itoa(remaining))
		if !allowed {
			c.Header("Retry-After", "60")
			abort(c, http.StatusTooManyRequests, "rate_limited", "请求过于频繁，请稍后重试")
			return
		}
		c.Next()
	}
}

func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Remaining")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func GetPrincipal(c *gin.Context) (service.Principal, bool) {
	value, exists := c.Get(PrincipalKey)
	if !exists {
		return service.Principal{}, false
	}
	principal, ok := value.(service.Principal)
	return principal, ok
}
func GetRequestID(c *gin.Context) string {
	value, _ := c.Get(RequestIDKey)
	id, _ := value.(string)
	return id
}

func (l *limiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	bucket := l.buckets[key]
	if bucket.window.IsZero() || now.Sub(bucket.window) >= time.Minute {
		bucket = rateBucket{window: now, count: 0}
	}
	if bucket.count >= l.limit {
		l.buckets[key] = bucket
		return false, 0
	}
	bucket.count++
	l.buckets[key] = bucket
	if now.Sub(l.lastCleanup) > 5*time.Minute {
		for item, value := range l.buckets {
			if now.Sub(value.window) > 2*time.Minute {
				delete(l.buckets, item)
			}
		}
		l.lastCleanup = now
	}
	return true, l.limit - bucket.count
}

func newRequestID() string {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		return "req-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "req-" + hex.EncodeToString(data)
}
func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message}, "request_id": GetRequestID(c)})
}
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 12)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
