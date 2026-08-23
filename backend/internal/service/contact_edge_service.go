package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
)

type ContactEdgeService struct {
	edges  repository.ContactEdgeRepository
	routes repository.RouteRepository
}

func NewContactEdgeService(edges repository.ContactEdgeRepository, routes repository.RouteRepository) *ContactEdgeService {
	return &ContactEdgeService{edges: edges, routes: routes}
}

func (s *ContactEdgeService) Create(ctx context.Context, request dto.CreateContactEdgeRequest, actor Principal, requestID string) (model.ContactEdge, error) {
	from, to, err := s.validateEndpoints(ctx, request.RouteID, request.FromStepCode, request.ToStepCode)
	if err != nil {
		return model.ContactEdge{}, err
	}
	existing, err := s.edges.ForRoute(ctx, request.RouteID)
	if err != nil {
		return model.ContactEdge{}, err
	}
	for _, edge := range existing {
		if edge.FromStepCode == from && edge.ToStepCode == to && edge.ContactType == request.ContactType {
			return model.ContactEdge{}, NewError(http.StatusConflict, "duplicate_edge", "相同方向和接触类型的边已存在", nil)
		}
	}
	edge := model.ContactEdge{RouteID: request.RouteID, FromStepCode: from, ToStepCode: to, ContactType: request.ContactType, SharedEquipment: strings.TrimSpace(request.SharedEquipment), CleaningFactor: request.CleaningFactor, CarryoverProbability: request.CarryoverProbability, EvidenceNote: strings.TrimSpace(request.EvidenceNote), Enabled: request.Enabled, Version: 1, CreatedBy: actor.ID}
	if err := s.edges.Create(ctx, &edge, AuditScope(actor, requestID)); err != nil {
		return model.ContactEdge{}, err
	}
	return edge, nil
}

func (s *ContactEdgeService) Get(ctx context.Context, id uint) (model.ContactEdge, error) {
	edge, err := s.edges.Get(ctx, id)
	if err != nil {
		return model.ContactEdge{}, fmt.Errorf("get contact edge: %v", err)
	}
	return edge, nil
}
func (s *ContactEdgeService) List(ctx context.Context, query dto.ContactEdgeQuery) ([]model.ContactEdge, int64, error) {
	return s.edges.List(ctx, query)
}

func (s *ContactEdgeService) Update(ctx context.Context, id uint, request dto.UpdateContactEdgeRequest, actor Principal, requestID string) (model.ContactEdge, error) {
	current, err := s.edges.Get(ctx, id)
	if err != nil {
		return model.ContactEdge{}, fmt.Errorf("load contact edge: %v", err)
	}
	edge := model.ContactEdge{ID: id, RouteID: current.RouteID, FromStepCode: current.FromStepCode, ToStepCode: current.ToStepCode, ContactType: request.ContactType, SharedEquipment: strings.TrimSpace(request.SharedEquipment), CleaningFactor: request.CleaningFactor, CarryoverProbability: request.CarryoverProbability, EvidenceNote: strings.TrimSpace(request.EvidenceNote), Enabled: request.Enabled}
	if err := s.edges.Update(ctx, &edge, request.ExpectedVersion, AuditScope(actor, requestID)); err != nil {
		return model.ContactEdge{}, err
	}
	return edge, nil
}

func (s *ContactEdgeService) validateEndpoints(ctx context.Context, routeID uint, fromInput, toInput string) (string, string, error) {
	route, err := s.routes.Get(ctx, routeID)
	if err != nil {
		return "", "", err
	}
	steps, err := DecodeRouteSteps(route)
	if err != nil {
		return "", "", err
	}
	from, to := strings.ToUpper(strings.TrimSpace(fromInput)), strings.ToUpper(strings.TrimSpace(toInput))
	codes := make(map[string]bool, len(steps))
	for _, step := range steps {
		codes[step.StepCode] = true
	}
	if from == to {
		return "", "", NewError(http.StatusBadRequest, "self_loop", "接触边不能连接同一步骤", nil)
	}
	if !codes[from] || !codes[to] {
		return "", "", NewError(http.StatusBadRequest, "unknown_step", "接触边端点必须属于所选路线", nil)
	}
	return from, to, nil
}
