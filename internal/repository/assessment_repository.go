package repository

import (
	"context"
	"fmt"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AssessmentRepository interface {
	Create(context.Context, *model.AssessmentRun, AuditContext) error
	Get(context.Context, uint) (model.AssessmentRun, error)
	List(context.Context, dto.AssessmentQuery) ([]model.AssessmentRun, int64, error)
	Summary(context.Context) (dto.AssessmentSummary, error)
	BeginCalculation(context.Context, uint, AuditContext) error
	ResetCalculation(context.Context, uint, string, AuditContext) error
	CompleteCalculation(context.Context, uint, datatypes.JSON, datatypes.JSON, datatypes.JSON, constants.RiskLevel, string, AuditContext) error
	Review(context.Context, uint, constants.AssessmentStatus, uint, string, AuditContext) error
}

type assessmentRepository struct{ db *gorm.DB }

func NewAssessmentRepository(db *gorm.DB) AssessmentRepository { return &assessmentRepository{db: db} }

func (r *assessmentRepository) Create(ctx context.Context, run *model.AssessmentRun, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("create assessment run: %w", err)
		}
		audit, err := makeAudit(scope, "assessment.queued", "assessment_run", run.ID, "", assessmentSummary(*run), map[string]any{"route_id": run.RouteID})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit assessment creation: %w", err)
		}
		return nil
	})
}

func (r *assessmentRepository) Get(ctx context.Context, id uint) (model.AssessmentRun, error) {
	var run model.AssessmentRun
	if err := r.db.WithContext(ctx).First(&run, id).Error; err != nil {
		return model.AssessmentRun{}, fmt.Errorf("get assessment run: %w", err)
	}
	return run, nil
}

func (r *assessmentRepository) List(ctx context.Context, query dto.AssessmentQuery) ([]model.AssessmentRun, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.AssessmentRun{})
	if query.RouteID > 0 {
		db = db.Where("route_id = ?", query.RouteID)
	}
	if query.Status != "" {
		db = db.Where("assessment_status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count assessment runs: %w", err)
	}
	page, size := pageValues(query.Page, query.PageSize)
	var runs []model.AssessmentRun
	if err := db.Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("list assessment runs: %w", err)
	}
	return runs, total, nil
}

func (r *assessmentRepository) Summary(ctx context.Context) (dto.AssessmentSummary, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&model.AssessmentRun{}).Select("assessment_status AS status, COUNT(*) AS count").Group("assessment_status").Scan(&rows).Error; err != nil {
		return dto.AssessmentSummary{}, fmt.Errorf("summarize assessment runs: %w", err)
	}
	result := dto.AssessmentSummary{ByStatus: make(map[string]int64)}
	for _, item := range rows {
		result.ByStatus[item.Status] = item.Count
		result.Total += item.Count
		if item.Status == string(constants.AssessmentPendingReview) {
			result.PendingReview = item.Count
		}
	}
	return result, nil
}

func (r *assessmentRepository) BeginCalculation(ctx context.Context, id uint, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AssessmentRun{}).Where("id = ? AND assessment_status = ?", id, constants.AssessmentQueued).Update("assessment_status", constants.AssessmentCalculating)
		if result.Error != nil {
			return fmt.Errorf("begin assessment calculation: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("begin assessment calculation: %w", ErrStateConflict)
		}
		audit, err := makeAudit(scope, "assessment.calculating", "assessment_run", id, string(constants.AssessmentQueued), string(constants.AssessmentCalculating), map[string]any{})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit calculation start: %w", err)
		}
		return nil
	})
}

func (r *assessmentRepository) ResetCalculation(ctx context.Context, id uint, reason string, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AssessmentRun{}).Where("id = ? AND assessment_status = ?", id, constants.AssessmentCalculating).Update("assessment_status", constants.AssessmentQueued)
		if result.Error != nil {
			return fmt.Errorf("reset assessment calculation: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("reset assessment calculation: %w", ErrStateConflict)
		}
		audit, err := makeAudit(scope, "assessment.calculation_failed", "assessment_run", id, string(constants.AssessmentCalculating), string(constants.AssessmentQueued), map[string]any{"reason": reason})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit calculation reset: %w", err)
		}
		return nil
	})
}

func (r *assessmentRepository) CompleteCalculation(ctx context.Context, id uint, snapshot, matrix, riskItems datatypes.JSON, highest constants.RiskLevel, algorithm string, scope AuditContext) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		updates := map[string]any{"assessment_status": constants.AssessmentPendingReview, "input_snapshot_json": snapshot, "matrix_json": matrix, "risk_items_json": riskItems, "highest_risk_level": highest, "algorithm_version": algorithm, "completed_at": &now}
		result := tx.Model(&model.AssessmentRun{}).Where("id = ? AND assessment_status = ?", id, constants.AssessmentCalculating).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("complete assessment calculation: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("complete assessment calculation: %w", ErrStateConflict)
		}
		audit, err := makeAudit(scope, "assessment.pending_review", "assessment_run", id, string(constants.AssessmentCalculating), string(constants.AssessmentPendingReview), map[string]any{"highest_risk_level": highest, "algorithm_version": algorithm})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit calculation completion: %w", err)
		}
		return nil
	})
}

func (r *assessmentRepository) Review(ctx context.Context, id uint, target constants.AssessmentStatus, reviewerID uint, reason string, scope AuditContext) error {
	if target != constants.AssessmentAccepted && target != constants.AssessmentRejected {
		return fmt.Errorf("review target %q: %w", target, ErrStateConflict)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		updates := map[string]any{"assessment_status": target, "reviewed_by": reviewerID, "review_reason": reason, "reviewed_at": &now}
		result := tx.Model(&model.AssessmentRun{}).Where("id = ? AND assessment_status = ?", id, constants.AssessmentPendingReview).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("review assessment: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("review assessment: %w", ErrStateConflict)
		}
		audit, err := makeAudit(scope, "assessment.reviewed", "assessment_run", id, string(constants.AssessmentPendingReview), string(target), map[string]any{"reviewer_id": reviewerID, "reason": reason})
		if err != nil {
			return err
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit assessment review: %w", err)
		}
		return nil
	})
}

func assessmentSummary(run model.AssessmentRun) string {
	return fmt.Sprintf("route=%d status=%s highest=%s algorithm=%s", run.RouteID, run.AssessmentStatus, run.HighestRiskLevel, run.AlgorithmVersion)
}
