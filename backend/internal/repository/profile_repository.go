package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(context.Context, *model.AllergenProfile, AuditContext) error
	Get(context.Context, uint) (model.AllergenProfile, error)
	List(context.Context, dto.ProfileQuery) ([]model.AllergenProfile, int64, error)
	Update(context.Context, *model.AllergenProfile, uint, AuditContext) error
	Usage(context.Context, uint) ([]dto.ProfileUsage, error)
	GetMany(context.Context, []uint) ([]model.AllergenProfile, error)
}

type profileRepository struct{ db *gorm.DB }

func NewProfileRepository(db *gorm.DB) ProfileRepository { return &profileRepository{db: db} }

func (r *profileRepository) Create(ctx context.Context, profile *model.AllergenProfile, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(profile).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				return fmt.Errorf("create profile: %w", ErrDuplicate)
			}
			return fmt.Errorf("create profile: %w", err)
		}
		audit, err := makeAudit(scope, "profile.created", "allergen_profile", profile.ID, "", profileSummary(*profile), map[string]any{"version": profile.Version})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit profile creation: %w", err)
		}
		return nil
	})
}

func (r *profileRepository) Get(ctx context.Context, id uint) (model.AllergenProfile, error) {
	var profile model.AllergenProfile
	if err := r.db.WithContext(ctx).First(&profile, id).Error; err != nil {
		return model.AllergenProfile{}, fmt.Errorf("get allergen profile: %w", err)
	}
	return profile, nil
}

func (r *profileRepository) List(ctx context.Context, query dto.ProfileQuery) ([]model.AllergenProfile, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.AllergenProfile{})
	if query.Search != "" {
		like := "%" + strings.ToLower(query.Search) + "%"
		db = db.Where("LOWER(profile_code) LIKE ? OR LOWER(material_name) LIKE ?", like, like)
	}
	if query.Status != "" {
		db = db.Where("profile_status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count allergen profiles: %w", err)
	}
	page, size := pageValues(query.Page, query.PageSize)
	var profiles []model.AllergenProfile
	if err := db.Order("updated_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&profiles).Error; err != nil {
		return nil, 0, fmt.Errorf("list allergen profiles: %w", err)
	}
	return profiles, total, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *model.AllergenProfile, expected uint, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.AllergenProfile
		if err := tx.First(&before, profile.ID).Error; err != nil {
			return fmt.Errorf("load profile before update: %w", err)
		}
		updates := map[string]any{"material_name": profile.MaterialName, "allergens_json": profile.AllergensJSON, "source_type": profile.SourceType, "supplier_statement_date": profile.SupplierStatementDate, "profile_status": profile.ProfileStatus, "version": gorm.Expr("version + 1")}
		result := tx.Model(&model.AllergenProfile{}).Where("id = ? AND version = ?", profile.ID, expected).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update allergen profile: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("update allergen profile: %w", ErrVersionConflict)
		}
		if err := markAllRunsStale(tx); err != nil {
			return err
		}
		if err := tx.First(profile, profile.ID).Error; err != nil {
			return fmt.Errorf("reload allergen profile: %w", err)
		}
		audit, err := makeAudit(scope, "profile.versioned", "allergen_profile", profile.ID, profileSummary(before), profileSummary(*profile), map[string]any{"from_version": expected, "to_version": profile.Version, "assessments_staled": true})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit profile update: %w", err)
		}
		return nil
	})
}

func (r *profileRepository) Usage(ctx context.Context, profileID uint) ([]dto.ProfileUsage, error) {
	var routes []model.ProcessRoute
	if err := r.db.WithContext(ctx).Order("route_code").Find(&routes).Error; err != nil {
		return nil, fmt.Errorf("list routes for profile usage: %w", err)
	}
	result := make([]dto.ProfileUsage, 0)
	for _, route := range routes {
		var steps []dto.RouteStep
		if err := json.Unmarshal(route.OrderedStepsJSON, &steps); err != nil {
			return nil, fmt.Errorf("decode steps for route %d: %w", route.ID, err)
		}
		for _, step := range steps {
			if step.ProfileID == profileID {
				result = append(result, dto.ProfileUsage{RouteID: route.ID, RouteCode: route.RouteCode, ProductName: route.ProductName, RouteVersion: route.Version})
				break
			}
		}
	}
	return result, nil
}

func (r *profileRepository) GetMany(ctx context.Context, ids []uint) ([]model.AllergenProfile, error) {
	if len(ids) == 0 {
		return []model.AllergenProfile{}, nil
	}
	var profiles []model.AllergenProfile
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("get allergen profiles: %w", err)
	}
	return profiles, nil
}

func profileSummary(profile model.AllergenProfile) string {
	return fmt.Sprintf("code=%s material=%s status=%s version=%d allergens=%s", profile.ProfileCode, profile.MaterialName, profile.ProfileStatus, profile.Version, string(profile.AllergensJSON))
}
