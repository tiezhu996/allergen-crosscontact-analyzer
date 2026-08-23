package repository

import (
	"context"
	"fmt"
	"strconv"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/gorm"
)

type ContactEdgeRepository interface {
	Create(context.Context, *model.ContactEdge, AuditContext) error
	Get(context.Context, uint) (model.ContactEdge, error)
	List(context.Context, dto.ContactEdgeQuery) ([]model.ContactEdge, int64, error)
	ForRoute(context.Context, uint) ([]model.ContactEdge, error)
	Update(context.Context, *model.ContactEdge, uint, AuditContext) error
}

type contactEdgeRepository struct{ db *gorm.DB }

func NewContactEdgeRepository(db *gorm.DB) ContactEdgeRepository {
	return &contactEdgeRepository{db: db}
}

func (r *contactEdgeRepository) Create(ctx context.Context, edge *model.ContactEdge, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(edge).Error; err != nil {
			return fmt.Errorf("create contact edge: %w", err)
		}
		if err := markRouteRunsStale(tx, edge.RouteID, "contact_edge_inputs_changed", scope); err != nil {
			return err
		}
		audit, err := makeAudit(scope, "contact_edge.created", "contact_edge", edge.ID, "", edgeSummary(*edge), map[string]any{"route_id": edge.RouteID, "version": edge.Version, "assessments_staled": true})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit edge creation: %w", err)
		}
		return nil
	})
}

func (r *contactEdgeRepository) Get(ctx context.Context, id uint) (model.ContactEdge, error) {
	var edge model.ContactEdge
	if err := r.db.WithContext(ctx).First(&edge, id).Error; err != nil {
		return model.ContactEdge{}, fmt.Errorf("get contact edge: %w", err)
	}
	return edge, nil
}

func (r *contactEdgeRepository) List(ctx context.Context, query dto.ContactEdgeQuery) ([]model.ContactEdge, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.ContactEdge{})
	if query.RouteID > 0 {
		db = db.Where("route_id = ?", query.RouteID)
	}
	if query.Enabled != "" {
		enabled, _ := strconv.ParseBool(query.Enabled)
		db = db.Where("enabled = ?", enabled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count contact edges: %w", err)
	}
	page, size := pageValues(query.Page, query.PageSize)
	var edges []model.ContactEdge
	if err := db.Order("route_id, from_step_code, to_step_code, id").Offset((page - 1) * size).Limit(size).Find(&edges).Error; err != nil {
		return nil, 0, fmt.Errorf("list contact edges: %w", err)
	}
	return edges, total, nil
}

func (r *contactEdgeRepository) ForRoute(ctx context.Context, routeID uint) ([]model.ContactEdge, error) {
	var edges []model.ContactEdge
	if err := r.db.WithContext(ctx).Where("route_id = ?", routeID).Order("id").Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("list route contact edges: %w", err)
	}
	return edges, nil
}

func (r *contactEdgeRepository) Update(ctx context.Context, edge *model.ContactEdge, expected uint, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.ContactEdge
		if err := tx.First(&before, edge.ID).Error; err != nil {
			return fmt.Errorf("load contact edge before update: %w", err)
		}
		updates := map[string]any{"contact_type": edge.ContactType, "shared_equipment": edge.SharedEquipment, "cleaning_factor": edge.CleaningFactor, "carryover_probability": edge.CarryoverProbability, "evidence_note": edge.EvidenceNote, "enabled": edge.Enabled, "version": gorm.Expr("version + 1")}
		result := tx.Model(&model.ContactEdge{}).Where("id = ? AND version = ?", edge.ID, expected).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update contact edge: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("update contact edge: %w", ErrVersionConflict)
		}
		if err := markRouteRunsStale(tx, before.RouteID, "contact_edge_inputs_changed", scope); err != nil {
			return err
		}
		if err := tx.First(edge, edge.ID).Error; err != nil {
			return fmt.Errorf("reload contact edge: %w", err)
		}
		audit, err := makeAudit(scope, "contact_edge.versioned", "contact_edge", edge.ID, edgeSummary(before), edgeSummary(*edge), map[string]any{"route_id": edge.RouteID, "from_version": expected, "to_version": edge.Version, "assessments_staled": true})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit edge update: %w", err)
		}
		return nil
	})
}

func edgeSummary(edge model.ContactEdge) string {
	return fmt.Sprintf("route=%d %s->%s type=%s equipment=%s cleaning=%.4f carryover=%.4f enabled=%s version=%d", edge.RouteID, edge.FromStepCode, edge.ToStepCode, edge.ContactType, edge.SharedEquipment, edge.CleaningFactor, edge.CarryoverProbability, strconv.FormatBool(edge.Enabled), edge.Version)
}
