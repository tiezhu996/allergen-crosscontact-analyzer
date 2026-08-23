package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"gorm.io/datatypes"
)

type RouteService struct {
	routes   repository.RouteRepository
	profiles repository.ProfileRepository
}

func NewRouteService(routes repository.RouteRepository, profiles repository.ProfileRepository) *RouteService {
	return &RouteService{routes: routes, profiles: profiles}
}

func (s *RouteService) Create(ctx context.Context, request dto.CreateRouteRequest, actor Principal, requestID string) (model.ProcessRoute, error) {
	code, err := normalizeIdentifier(request.RouteCode)
	if err != nil {
		return model.ProcessRoute{}, err
	}
	steps, declared, err := s.encodeInputs(ctx, request.OrderedSteps, request.DeclaredAllergens, request.RouteStatus)
	if err != nil {
		return model.ProcessRoute{}, err
	}
	route := model.ProcessRoute{RouteCode: code, ProductName: strings.TrimSpace(request.ProductName), OrderedStepsJSON: steps, DeclaredAllergensJSON: declared, RouteStatus: request.RouteStatus, Version: 1, OwnerID: actor.ID}
	if err := s.routes.Create(ctx, &route, AuditScope(actor, requestID)); err != nil {
		return model.ProcessRoute{}, err
	}
	return route, nil
}

func (s *RouteService) Get(ctx context.Context, id uint) (model.ProcessRoute, error) {
	return s.routes.Get(ctx, id)
}
func (s *RouteService) List(ctx context.Context, query dto.RouteQuery) ([]model.ProcessRoute, int64, error) {
	return s.routes.List(ctx, query)
}

func (s *RouteService) Update(ctx context.Context, id uint, request dto.UpdateRouteRequest, actor Principal, requestID string) (model.ProcessRoute, error) {
	steps, declared, err := s.encodeInputs(ctx, request.OrderedSteps, request.DeclaredAllergens, request.RouteStatus)
	if err != nil {
		return model.ProcessRoute{}, err
	}
	route := model.ProcessRoute{ID: id, ProductName: strings.TrimSpace(request.ProductName), OrderedStepsJSON: steps, DeclaredAllergensJSON: declared, RouteStatus: request.RouteStatus}
	if err := s.routes.Update(ctx, &route, request.ExpectedVersion, AuditScope(actor, requestID)); err != nil {
		return model.ProcessRoute{}, err
	}
	return route, nil
}

func (s *RouteService) encodeInputs(ctx context.Context, input []dto.RouteStep, declaredInput []string, status string) (datatypes.JSON, datatypes.JSON, error) {
	steps := make([]dto.RouteStep, len(input))
	copy(steps, input)
	seenCodes := make(map[string]bool)
	profileIDs := make([]uint, 0, len(steps))
	profileSet := make(map[uint]bool)
	for index := range steps {
		code, err := normalizeIdentifier(steps[index].StepCode)
		if err != nil {
			return nil, nil, NewError(http.StatusBadRequest, "invalid_step_code", fmt.Sprintf("第 %d 个步骤代码无效", index+1), err)
		}
		steps[index].StepCode = code
		steps[index].StepName = strings.TrimSpace(steps[index].StepName)
		if seenCodes[steps[index].StepCode] {
			return nil, nil, NewError(http.StatusBadRequest, "duplicate_step", fmt.Sprintf("步骤代码 %s 重复", steps[index].StepCode), nil)
		}
		seenCodes[steps[index].StepCode] = true
		if !profileSet[steps[index].ProfileID] {
			profileSet[steps[index].ProfileID] = true
			profileIDs = append(profileIDs, steps[index].ProfileID)
		}
	}
	profiles, err := s.profiles.GetMany(ctx, profileIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(profiles) != len(profileIDs) {
		return nil, nil, NewError(http.StatusBadRequest, "profile_missing", "路线引用了不存在的过敏原谱", nil)
	}
	if status == "active" {
		for _, profile := range profiles {
			if profile.ProfileStatus != "active" {
				return nil, nil, NewError(http.StatusBadRequest, "profile_inactive", fmt.Sprintf("激活路线不能引用非 active 谱 %s", profile.ProfileCode), nil)
			}
		}
	}
	declaredValues := normalizeStrings(declaredInput)
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, nil, fmt.Errorf("encode ordered steps: %w", err)
	}
	declaredJSON, err := json.Marshal(declaredValues)
	if err != nil {
		return nil, nil, fmt.Errorf("encode declared allergens: %w", err)
	}
	return datatypes.JSON(stepsJSON), datatypes.JSON(declaredJSON), nil
}

func DecodeRouteSteps(route model.ProcessRoute) ([]dto.RouteStep, error) {
	var steps []dto.RouteStep
	if err := json.Unmarshal(route.OrderedStepsJSON, &steps); err != nil {
		return nil, fmt.Errorf("decode route steps: %w", err)
	}
	if len(steps) < 2 {
		return nil, NewError(http.StatusUnprocessableEntity, "invalid_graph", "路线至少需要两个有效步骤", nil)
	}
	return steps, nil
}

func DecodeDeclared(route model.ProcessRoute) ([]string, error) {
	var allergens []string
	if err := json.Unmarshal(route.DeclaredAllergensJSON, &allergens); err != nil {
		return nil, fmt.Errorf("decode declared allergens: %w", err)
	}
	return normalizeStrings(allergens), nil
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]string)
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean != "" {
			key := strings.ToLower(clean)
			if _, exists := seen[key]; !exists {
				seen[key] = clean
			}
		}
	}
	result := make([]string, 0, len(seen))
	for _, value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
