package service

import (
	"context"
	"testing"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
)

type principalRepoStub struct{ user model.User }

func (r *principalRepoStub) UserByUsername(context.Context, string) (model.User, error) {
	return r.user, nil
}
func (r *principalRepoStub) UserByID(context.Context, uint) (model.User, error) { return r.user, nil }
func (r *principalRepoStub) ListAudit(context.Context, dto.AuditQuery) ([]model.AuditEvent, int64, error) {
	return nil, 0, nil
}
func (r *principalRepoStub) LatestAudit(context.Context, string, uint) (model.AuditEvent, error) {
	return model.AuditEvent{}, nil
}

func TestCurrentPrincipalUsesCurrentAccountState(t *testing.T) {
	stub := &principalRepoStub{user: model.User{ID: 7, Username: "analyst", DisplayName: "Analyst", Role: constants.RoleQualityAnalyst, Active: true}}
	service := NewSupportService(stub, config.Config{JWTSecret: "secret", JWTExpiry: time.Hour})
	principal, err := service.CurrentPrincipal(context.Background(), Principal{ID: 7, Role: constants.RoleAdmin})
	if err != nil {
		t.Fatalf("current principal: %v", err)
	}
	if principal.Role != constants.RoleQualityAnalyst {
		t.Fatalf("role = %q, want %q", principal.Role, constants.RoleQualityAnalyst)
	}

	stub.user.Active = false
	if _, err := service.CurrentPrincipal(context.Background(), Principal{ID: 7}); err == nil {
		t.Fatal("inactive user unexpectedly authenticated")
	}
}
