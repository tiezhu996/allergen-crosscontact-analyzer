package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/analyzer"
	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"gorm.io/datatypes"
)

type AssessmentService struct {
	runs       repository.AssessmentRepository
	routes     repository.RouteRepository
	profiles   repository.ProfileRepository
	edges      repository.ContactEdgeRepository
	maxDepth   int
	thresholds analyzer.ThresholdSnapshot
	algorithm  string
}

type queuedAssessmentSnapshot struct {
	RouteID             uint `json:"route_id"`
	RouteVersionAtQueue uint `json:"route_version_at_queue"`
}

func NewAssessmentService(runs repository.AssessmentRepository, routes repository.RouteRepository, profiles repository.ProfileRepository, edges repository.ContactEdgeRepository, cfg config.Config) (*AssessmentService, error) {
	thresholds, err := analyzer.NewThresholdSnapshot(cfg.Thresholds)
	if err != nil {
		return nil, fmt.Errorf("initialize thresholds: %w", err)
	}
	return &AssessmentService{runs: runs, routes: routes, profiles: profiles, edges: edges, maxDepth: cfg.MaxPropagationDepth, thresholds: thresholds, algorithm: "weighted-path-v1/" + thresholds.Version}, nil
}

func (s *AssessmentService) Preview(ctx context.Context, routeID uint) (analyzer.Result, error) {
	result, _, err := s.compute(ctx, routeID)
	return result, err
}

func (s *AssessmentService) Create(ctx context.Context, request dto.CreateAssessmentRequest, actor Principal, requestID string) (model.AssessmentRun, error) {
	route, err := s.routes.Get(ctx, request.RouteID)
	if err != nil {
		return model.AssessmentRun{}, err
	}
	if route.RouteStatus != "active" {
		return model.AssessmentRun{}, NewError(http.StatusConflict, "route_inactive", "只有 active 路线可以提交评估", nil)
	}
	queuedSnapshot, err := json.Marshal(map[string]any{"route_id": route.ID, "route_version_at_queue": route.Version, "queued_at": time.Now().UTC(), "threshold_version": s.thresholds.Version})
	if err != nil {
		return model.AssessmentRun{}, fmt.Errorf("encode queued snapshot: %w", err)
	}
	run := model.AssessmentRun{RouteID: route.ID, AssessmentStatus: constants.AssessmentQueued, InputSnapshotJSON: datatypes.JSON(queuedSnapshot), MatrixJSON: datatypes.JSON([]byte("[]")), RiskItemsJSON: datatypes.JSON([]byte("[]")), HighestRiskLevel: constants.RiskLow, AlgorithmVersion: s.algorithm, CreatedBy: actor.ID}
	if err := s.runs.Create(ctx, &run, AuditScope(actor, requestID)); err != nil {
		return model.AssessmentRun{}, err
	}
	return run, nil
}

func (s *AssessmentService) Run(ctx context.Context, id uint, actor Principal, requestID string) (model.AssessmentRun, error) {
	if err := s.runs.BeginCalculation(ctx, id, AuditScope(actor, requestID)); err != nil {
		return model.AssessmentRun{}, err
	}
	// From here the run is in the calculating state. Any failure before
	// CompleteCalculation succeeds must roll the run back to queued so it can be
	// retried instead of being stranded in calculating forever. rollback is the
	// single chokepoint: it resets state and returns the original caller-facing
	// error so the audit trail records the cause without masking it.
	rollback := func(err error) (model.AssessmentRun, error) {
		s.resetAfterFailure(ctx, id, err, actor, requestID)
		return model.AssessmentRun{}, err
	}
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return rollback(err)
	}
	var queued queuedAssessmentSnapshot
	if err := json.Unmarshal(run.InputSnapshotJSON, &queued); err != nil || queued.RouteID != run.RouteID || queued.RouteVersionAtQueue == 0 {
		return rollback(NewError(http.StatusUnprocessableEntity, "assessment_snapshot_invalid", "评估排队快照无效，请重新提交评估", err))
	}
	route, err := s.routes.Get(ctx, run.RouteID)
	if err != nil {
		return rollback(err)
	}
	if route.Version != queued.RouteVersionAtQueue {
		return rollback(NewError(http.StatusConflict, "route_version_conflict", "路线在评估排队后已变化，请重新提交评估", nil))
	}
	result, snapshot, err := s.computeRoute(ctx, route)
	if err != nil {
		return rollback(err)
	}
	matrixJSON, err := json.Marshal(result.Matrix)
	if err != nil {
		return rollback(fmt.Errorf("encode assessment matrix: %w", err))
	}
	riskJSON, err := json.Marshal(result.RiskItems)
	if err != nil {
		return rollback(fmt.Errorf("encode assessment risk items: %w", err))
	}
	if err := s.runs.CompleteCalculation(ctx, id, snapshot, datatypes.JSON(matrixJSON), datatypes.JSON(riskJSON), result.HighestRiskLevel, s.algorithm, AuditScope(actor, requestID)); err != nil {
		return rollback(err)
	}
	return s.runs.Get(ctx, id)
}

func (s *AssessmentService) Review(ctx context.Context, id uint, request dto.ReviewAssessmentRequest, actor Principal, requestID string) (model.AssessmentRun, error) {
	if !CanReview(actor.Role) {
		return model.AssessmentRun{}, NewError(http.StatusForbidden, "forbidden", "仅 reviewer 或 admin 可复核评估", nil)
	}
	target := constants.AssessmentStatus(request.Decision)
	if target != constants.AssessmentAccepted && target != constants.AssessmentRejected {
		return model.AssessmentRun{}, NewError(http.StatusBadRequest, "invalid_decision", "复核决定无效", nil)
	}
	if err := s.runs.Review(ctx, id, target, actor.ID, request.Reason, AuditScope(actor, requestID)); err != nil {
		return model.AssessmentRun{}, err
	}
	return s.runs.Get(ctx, id)
}

func (s *AssessmentService) Get(ctx context.Context, id uint) (model.AssessmentRun, error) {
	return s.runs.Get(ctx, id)
}
func (s *AssessmentService) List(ctx context.Context, query dto.AssessmentQuery) ([]model.AssessmentRun, int64, error) {
	return s.runs.List(ctx, query)
}
func (s *AssessmentService) Summary(ctx context.Context) (dto.AssessmentSummary, error) {
	return s.runs.Summary(ctx)
}

func (s *AssessmentService) compute(ctx context.Context, routeID uint) (analyzer.Result, datatypes.JSON, error) {
	route, err := s.routes.Get(ctx, routeID)
	if err != nil {
		return analyzer.Result{}, nil, err
	}
	return s.computeRoute(ctx, route)
}

func (s *AssessmentService) computeRoute(ctx context.Context, route model.ProcessRoute) (analyzer.Result, datatypes.JSON, error) {
	steps, err := DecodeRouteSteps(route)
	if err != nil {
		return analyzer.Result{}, nil, err
	}
	declared, err := DecodeDeclared(route)
	if err != nil {
		return analyzer.Result{}, nil, err
	}
	edges, err := s.edges.ForRoute(ctx, route.ID)
	if err != nil {
		return analyzer.Result{}, nil, err
	}
	ids := make([]uint, 0, len(steps))
	seen := make(map[uint]bool)
	for _, step := range steps {
		if !seen[step.ProfileID] {
			seen[step.ProfileID] = true
			ids = append(ids, step.ProfileID)
		}
	}
	profiles, err := s.profiles.GetMany(ctx, ids)
	if err != nil {
		return analyzer.Result{}, nil, err
	}
	if len(profiles) != len(ids) {
		return analyzer.Result{}, nil, NewError(http.StatusUnprocessableEntity, "profile_missing", "路线引用的过敏原谱已不可用", nil)
	}
	seeds := make(map[uint]analyzer.ProfileSeed, len(profiles))
	profileVersions := make([]map[string]any, 0, len(profiles))
	for _, profile := range profiles {
		var allergens []string
		if err := json.Unmarshal(profile.AllergensJSON, &allergens); err != nil {
			return analyzer.Result{}, nil, NewError(http.StatusUnprocessableEntity, "profile_json_invalid", "过敏原谱内容无法解析", err)
		}
		seeds[profile.ID] = analyzer.ProfileSeed{ProfileID: profile.ID, ProfileCode: profile.ProfileCode, MaterialName: profile.MaterialName, Version: profile.Version, Allergens: allergens}
		profileVersions = append(profileVersions, map[string]any{"id": profile.ID, "code": profile.ProfileCode, "version": profile.Version})
	}
	graph, err := analyzer.BuildGraph(steps, edges)
	if err != nil {
		return analyzer.Result{}, nil, NewError(http.StatusUnprocessableEntity, "graph_invalid", "接触图结构无效", err)
	}
	result, err := analyzer.Propagate(graph, seeds, declared, s.maxDepth, s.thresholds)
	if err != nil {
		return analyzer.Result{}, nil, NewError(http.StatusUnprocessableEntity, "propagation_failed", "风险传播计算失败", err)
	}
	sort.Slice(profileVersions, func(i, j int) bool { return profileVersions[i]["id"].(uint) < profileVersions[j]["id"].(uint) })
	edgeVersions := make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		edgeVersions = append(edgeVersions, map[string]any{"id": edge.ID, "version": edge.Version, "enabled": edge.Enabled})
	}
	snapshotValue := map[string]any{"captured_at": time.Now().UTC(), "route": map[string]any{"id": route.ID, "code": route.RouteCode, "version": route.Version, "steps": steps, "declared_allergens": declared}, "profiles": profileVersions, "contact_edges": edgeVersions, "thresholds": s.thresholds, "algorithm_version": s.algorithm, "max_depth": s.maxDepth, "cycles": result.Cycles}
	snapshot, err := json.Marshal(snapshotValue)
	if err != nil {
		return analyzer.Result{}, nil, fmt.Errorf("encode input snapshot: %w", err)
	}
	return result, datatypes.JSON(snapshot), nil
}

func (s *AssessmentService) resetAfterFailure(ctx context.Context, id uint, calculationErr error, actor Principal, requestID string) {
	_ = s.runs.ResetCalculation(ctx, id, calculationErr.Error(), AuditScope(actor, requestID))
}
