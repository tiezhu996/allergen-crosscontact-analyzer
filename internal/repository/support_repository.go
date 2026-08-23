package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrVersionConflict = errors.New("version conflict")
	ErrStateConflict   = errors.New("state conflict")
	ErrDuplicate       = errors.New("duplicate record")
)

type AuditContext struct {
	RequestID string
	ActorID   uint
	ActorName string
}

type Database struct{ DB *gorm.DB }

func Open(cfg config.Config) (*Database, error) {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("open database: unsupported driver %q", cfg.DBDriver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obtain sql database: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return &Database{DB: db}, nil
}

func (d *Database) Migrate(ctx context.Context) error {
	models := []any{&model.User{}, &model.AllergenProfile{}, &model.ProcessRoute{}, &model.ContactEdge{}, &model.AssessmentRun{}, &model.AuditEvent{}}
	if err := d.DB.WithContext(ctx).AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate schema: %w", err)
	}
	return nil
}

func (d *Database) Ping(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("obtain sql database: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("obtain sql database: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

func (d *Database) Seed(ctx context.Context) error {
	return d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := []struct {
			username, password, display string
			role                        constants.Role
		}{
			{"analyst", "Analyst#525", "质量分析员", constants.RoleQualityAnalyst},
			{"reviewer", "Reviewer#525", "复核负责人", constants.RoleReviewer},
			{"admin", "Admin#525Secure", "系统管理员", constants.RoleAdmin},
		}
		for _, seed := range users {
			hash, err := bcrypt.GenerateFromPassword([]byte(seed.password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash seed password: %w", err)
			}
			user := model.User{Username: seed.username, PasswordHash: string(hash), DisplayName: seed.display, Role: seed.role, Active: true}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "username"}}, DoNothing: true}).Create(&user).Error; err != nil {
				return fmt.Errorf("seed user %s: %w", seed.username, err)
			}
		}
		var analyst model.User
		if err := tx.Where("username = ?", "analyst").First(&analyst).Error; err != nil {
			return fmt.Errorf("load seed analyst: %w", err)
		}
		var count int64
		if err := tx.Model(&model.AllergenProfile{}).Count(&count).Error; err != nil {
			return fmt.Errorf("count profiles: %w", err)
		}
		if count > 0 {
			return nil
		}
		date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
		peanutJSON, _ := json.Marshal([]string{"Peanut"})
		milkJSON, _ := json.Marshal([]string{"Milk"})
		profiles := []model.AllergenProfile{
			{ProfileCode: "MAT-PEANUT", MaterialName: "Roasted peanut paste", AllergensJSON: datatypes.JSON(peanutJSON), SourceType: "supplier_statement", SupplierStatementDate: &date, ProfileStatus: "active", Version: 1, CreatedBy: analyst.ID},
			{ProfileCode: "MAT-MILK", MaterialName: "Milk powder", AllergensJSON: datatypes.JSON(milkJSON), SourceType: "supplier_statement", SupplierStatementDate: &date, ProfileStatus: "active", Version: 1, CreatedBy: analyst.ID},
		}
		if err := tx.Create(&profiles).Error; err != nil {
			return fmt.Errorf("seed profiles: %w", err)
		}
		steps, _ := json.Marshal([]dto.RouteStep{{StepCode: "MIX-01", StepName: "Primary mixing", ProfileID: profiles[0].ID}, {StepCode: "FILL-02", StepName: "Shared filler", ProfileID: profiles[1].ID}, {StepCode: "PACK-03", StepName: "Final packing", ProfileID: profiles[1].ID}})
		declared, _ := json.Marshal([]string{"Milk"})
		route := model.ProcessRoute{RouteCode: "RT-NUT-COOKIE", ProductName: "Nut cookie line study", OrderedStepsJSON: datatypes.JSON(steps), DeclaredAllergensJSON: datatypes.JSON(declared), RouteStatus: "active", Version: 1, OwnerID: analyst.ID}
		if err := tx.Create(&route).Error; err != nil {
			return fmt.Errorf("seed route: %w", err)
		}
		edges := []model.ContactEdge{
			{RouteID: route.ID, FromStepCode: "MIX-01", ToStepCode: "FILL-02", ContactType: "shared_line", SharedEquipment: "Filler manifold A", CleaningFactor: .45, CarryoverProbability: .82, EvidenceNote: "Validated wet-cleaning cycle WC-18; visual inspection recorded.", Enabled: true, Version: 1, CreatedBy: analyst.ID},
			{RouteID: route.ID, FromStepCode: "FILL-02", ToStepCode: "PACK-03", ContactType: "sequence", SharedEquipment: "Transfer belt 3", CleaningFactor: .25, CarryoverProbability: .58, EvidenceNote: "Dry clean instruction DC-07 and changeover checklist sampled.", Enabled: true, Version: 1, CreatedBy: analyst.ID},
		}
		if err := tx.Create(&edges).Error; err != nil {
			return fmt.Errorf("seed contact edges: %w", err)
		}
		return nil
	})
}

type SupportRepository interface {
	UserByUsername(context.Context, string) (model.User, error)
	UserByID(context.Context, uint) (model.User, error)
	ListAudit(context.Context, dto.AuditQuery) ([]model.AuditEvent, int64, error)
	LatestAudit(context.Context, string, uint) (model.AuditEvent, error)
}

type supportRepository struct{ db *gorm.DB }

func NewSupportRepository(db *gorm.DB) SupportRepository { return &supportRepository{db: db} }

func (r *supportRepository) UserByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(username))).First(&user).Error; err != nil {
		return model.User{}, fmt.Errorf("find user by username: %w", err)
	}
	return user, nil
}

func (r *supportRepository) UserByID(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return model.User{}, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func (r *supportRepository) ListAudit(ctx context.Context, query dto.AuditQuery) ([]model.AuditEvent, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.AuditEvent{})
	if query.Action != "" {
		db = db.Where("action = ?", query.Action)
	}
	if query.EntityType != "" {
		db = db.Where("entity_type = ?", query.EntityType)
	}
	if query.EntityID > 0 {
		db = db.Where("entity_id = ?", query.EntityID)
	}
	if query.RequestID != "" {
		db = db.Where("request_id = ?", query.RequestID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	page, size := pageValues(query.Page, query.PageSize)
	var events []model.AuditEvent
	if err := db.Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	return events, total, nil
}

func (r *supportRepository) LatestAudit(ctx context.Context, entityType string, entityID uint) (model.AuditEvent, error) {
	var event model.AuditEvent
	if err := r.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ?", entityType, entityID).Order("created_at DESC, id DESC").First(&event).Error; err != nil {
		return model.AuditEvent{}, fmt.Errorf("find latest audit event: %w", err)
	}
	return event, nil
}

func makeAudit(scope AuditContext, action, entityType string, entityID uint, before, after string, metadata any) (model.AuditEvent, error) {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return model.AuditEvent{}, fmt.Errorf("encode audit metadata: %w", err)
	}
	return model.AuditEvent{RequestID: scope.RequestID, ActorID: scope.ActorID, ActorName: scope.ActorName, Action: action, EntityType: entityType, EntityID: entityID, BeforeSummary: before, AfterSummary: after, MetadataJSON: encoded, CreatedAt: time.Now().UTC()}, nil
}

func markRouteRunsStale(tx *gorm.DB, routeID uint) error {
	statuses := []constants.AssessmentStatus{constants.AssessmentPendingReview, constants.AssessmentAccepted, constants.AssessmentRejected}
	if err := tx.Model(&model.AssessmentRun{}).Where("route_id = ? AND assessment_status IN ?", routeID, statuses).Update("assessment_status", constants.AssessmentStale).Error; err != nil {
		return fmt.Errorf("mark route assessments stale: %w", err)
	}
	return nil
}

func markAllRunsStale(tx *gorm.DB) error {
	statuses := []constants.AssessmentStatus{constants.AssessmentPendingReview, constants.AssessmentAccepted, constants.AssessmentRejected}
	if err := tx.Model(&model.AssessmentRun{}).Where("assessment_status IN ?", statuses).Update("assessment_status", constants.AssessmentStale).Error; err != nil {
		return fmt.Errorf("mark assessments stale: %w", err)
	}
	return nil
}

func pageValues(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	return page, size
}
