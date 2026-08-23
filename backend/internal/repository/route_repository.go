package repository

import (
	"context"
	"fmt"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/gorm"
)

type RouteRepository interface {
	Create(context.Context, *model.ProcessRoute, AuditContext) error
	Get(context.Context, uint) (model.ProcessRoute, error)
	List(context.Context, dto.RouteQuery) ([]model.ProcessRoute, int64, error)
	Update(context.Context, *model.ProcessRoute, uint, AuditContext) error
}

type routeRepository struct{ db *gorm.DB }

func NewRouteRepository(db *gorm.DB) RouteRepository { return &routeRepository{db: db} }

func (r *routeRepository) Create(ctx context.Context, route *model.ProcessRoute, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(route).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				return fmt.Errorf("create process route: %w", ErrDuplicate)
			}
			return fmt.Errorf("create process route: %w", err)
		}
		audit, err := makeAudit(scope, "route.created", "process_route", route.ID, "", routeSummary(*route), map[string]any{"version": route.Version})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit route creation: %w", err)
		}
		return nil
	})
}

func (r *routeRepository) Get(ctx context.Context, id uint) (model.ProcessRoute, error) {
	var route model.ProcessRoute
	if err := r.db.WithContext(ctx).First(&route, id).Error; err != nil {
		return model.ProcessRoute{}, fmt.Errorf("get process route: %w", err)
	}
	return route, nil
}

func (r *routeRepository) List(ctx context.Context, query dto.RouteQuery) ([]model.ProcessRoute, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.ProcessRoute{})
	if query.Search != "" {
		like := "%" + strings.ToLower(query.Search) + "%"
		db = db.Where("LOWER(route_code) LIKE ? OR LOWER(product_name) LIKE ?", like, like)
	}
	if query.Status != "" {
		db = db.Where("route_status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count process routes: %w", err)
	}
	page, size := pageValues(query.Page, query.PageSize)
	var routes []model.ProcessRoute
	if err := db.Order("updated_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&routes).Error; err != nil {
		return nil, 0, fmt.Errorf("list process routes: %w", err)
	}
	return routes, total, nil
}

func (r *routeRepository) Update(ctx context.Context, route *model.ProcessRoute, expected uint, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.ProcessRoute
		if err := tx.First(&before, route.ID).Error; err != nil {
			return fmt.Errorf("load route before update: %w", err)
		}
		updates := map[string]any{"product_name": route.ProductName, "ordered_steps_json": route.OrderedStepsJSON, "declared_allergens_json": route.DeclaredAllergensJSON, "route_status": route.RouteStatus, "version": gorm.Expr("version + 1")}
		result := tx.Model(&model.ProcessRoute{}).Where("id = ? AND version = ?", route.ID, expected).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update process route: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("update process route: %w", ErrVersionConflict)
		}
		if err := markRouteRunsStale(tx, route.ID); err != nil {
			return err
		}
		if err := tx.First(route, route.ID).Error; err != nil {
			return fmt.Errorf("reload process route: %w", err)
		}
		audit, err := makeAudit(scope, "route.versioned", "process_route", route.ID, routeSummary(before), routeSummary(*route), map[string]any{"from_version": expected, "to_version": route.Version, "assessments_staled": true})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit route update: %w", err)
		}
		return nil
	})
}

func routeSummary(route model.ProcessRoute) string {
	return fmt.Sprintf("code=%s product=%s status=%s version=%d steps=%s declared=%s", route.RouteCode, route.ProductName, route.RouteStatus, route.Version, string(route.OrderedStepsJSON), string(route.DeclaredAllergensJSON))
}
