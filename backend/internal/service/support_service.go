package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}
func (e *AppError) Unwrap() error { return e.Cause }

func NewError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}

func NormalizeError(err error) *AppError {
	if err == nil {
		return nil
	}
	var app *AppError
	if errors.As(err, &app) {
		return app
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewError(http.StatusNotFound, "not_found", "请求的资源不存在", err)
	case errors.Is(err, repository.ErrDuplicate), errors.Is(err, gorm.ErrDuplicatedKey):
		return NewError(http.StatusConflict, "duplicate", "唯一标识已存在", err)
	case errors.Is(err, repository.ErrVersionConflict):
		return NewError(http.StatusConflict, "version_conflict", "输入版本已变化，请刷新后重试", err)
	case errors.Is(err, repository.ErrStateConflict):
		return NewError(http.StatusConflict, "state_conflict", "当前状态不允许此操作", err)
	default:
		return NewError(http.StatusInternalServerError, "internal_error", "服务处理失败", err)
	}
}

type Principal struct {
	ID          uint           `json:"id"`
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	Role        constants.Role `json:"role"`
}

type authClaims struct {
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	Role        constants.Role `json:"role"`
	jwt.RegisteredClaims
}

type SupportService struct {
	repo   repository.SupportRepository
	secret []byte
	expiry time.Duration
}

func NewSupportService(repo repository.SupportRepository, cfg config.Config) *SupportService {
	return &SupportService{repo: repo, secret: []byte(cfg.JWTSecret), expiry: cfg.JWTExpiry}
}

func (s *SupportService) Login(ctx context.Context, request dto.LoginRequest) (dto.LoginResponse, error) {
	user, err := s.repo.UserByUsername(ctx, request.Username)
	if err != nil || !user.Active {
		return dto.LoginResponse{}, NewError(http.StatusUnauthorized, "invalid_credentials", "用户名或密码不正确", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return dto.LoginResponse{}, NewError(http.StatusUnauthorized, "invalid_credentials", "用户名或密码不正确", err)
	}
	now := time.Now().UTC()
	expires := now.Add(s.expiry)
	claims := authClaims{Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: fmt.Sprint(user.ID), Issuer: "food-allergen-crosscontact-analyzer", Audience: jwt.ClaimStrings{"food-allergen-console"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires), NotBefore: jwt.NewNumericDate(now.Add(-time.Minute))}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("sign login token: %w", err)
	}
	view := dto.UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role}
	return dto.LoginResponse{Token: signed, ExpiresAt: expires.Format(time.RFC3339), User: view}, nil
}

func (s *SupportService) ParseToken(raw string) (Principal, error) {
	claims := authClaims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return s.secret, nil
	}, jwt.WithAudience("food-allergen-console"), jwt.WithIssuer("food-allergen-crosscontact-analyzer"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Principal{}, NewError(http.StatusUnauthorized, "invalid_token", "登录凭证无效或已过期", err)
	}
	var id uint
	if _, err := fmt.Sscan(claims.Subject, &id); err != nil || id == 0 {
		return Principal{}, NewError(http.StatusUnauthorized, "invalid_token", "登录凭证主体无效", err)
	}
	if !claims.Role.Valid() {
		return Principal{}, NewError(http.StatusUnauthorized, "invalid_token", "登录凭证角色无效", nil)
	}
	return Principal{ID: id, Username: claims.Username, DisplayName: claims.DisplayName, Role: claims.Role}, nil
}

func (s *SupportService) CurrentUser(ctx context.Context, principal Principal) (dto.UserView, error) {
	user, err := s.repo.UserByID(ctx, principal.ID)
	if err != nil {
		return dto.UserView{}, err
	}
	if !user.Active {
		return dto.UserView{}, NewError(http.StatusUnauthorized, "inactive_user", "账号已停用", nil)
	}
	return dto.UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role}, nil
}

// CurrentPrincipal revalidates the account for every protected request so
// deactivation and role changes take effect before RBAC is evaluated.
func (s *SupportService) CurrentPrincipal(ctx context.Context, principal Principal) (Principal, error) {
	user, err := s.repo.UserByID(ctx, principal.ID)
	if err != nil {
		return Principal{}, NewError(http.StatusUnauthorized, "invalid_token", "登录账号不可用", err)
	}
	if !user.Active {
		return Principal{}, NewError(http.StatusUnauthorized, "inactive_user", "账号已停用", nil)
	}
	role := constants.Role(user.Role)
	if !role.Valid() {
		return Principal{}, NewError(http.StatusUnauthorized, "invalid_account_role", "登录账号角色无效", nil)
	}
	return Principal{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: role}, nil
}

func (s *SupportService) ListAudit(ctx context.Context, query dto.AuditQuery) ([]model.AuditEvent, int64, error) {
	return s.repo.ListAudit(ctx, query)
}

func (s *SupportService) VersionDiff(ctx context.Context, entityType string, id, version uint) (dto.VersionDiff, error) {
	allowed := map[string]bool{"process_route": true, "allergen_profile": true, "contact_edge": true, "assessment_run": true}
	if !allowed[entityType] {
		return dto.VersionDiff{}, NewError(http.StatusBadRequest, "invalid_entity_type", "不支持的实体类型", nil)
	}
	event, err := s.repo.LatestAudit(ctx, entityType, id)
	if err != nil {
		return dto.VersionDiff{}, err
	}
	latest := map[string]any{"action": event.Action, "before": event.BeforeSummary, "after": event.AfterSummary, "actor": event.ActorName, "request_id": event.RequestID, "created_at": event.CreatedAt}
	return dto.VersionDiff{EntityType: entityType, EntityID: id, Version: version, LatestAudit: latest}, nil
}

func AuditScope(principal Principal, requestID string) repository.AuditContext {
	return repository.AuditContext{RequestID: strings.TrimSpace(requestID), ActorID: principal.ID, ActorName: principal.DisplayName}
}

func CanReview(role constants.Role) bool {
	return role == constants.RoleReviewer || role == constants.RoleAdmin
}
func CanEdit(role constants.Role) bool {
	return role == constants.RoleQualityAnalyst || role == constants.RoleAdmin
}
