package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
	"food-allergen-crosscontact-analyzer/backend/internal/repository"
	"gorm.io/datatypes"
)

type ProfileService struct{ repo repository.ProfileRepository }

func NewProfileService(repo repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) Create(ctx context.Context, request dto.CreateProfileRequest, actor Principal, requestID string) (model.AllergenProfile, error) {
	code, err := normalizeIdentifier(request.ProfileCode)
	if err != nil {
		return model.AllergenProfile{}, err
	}
	allergens, err := encodeAllergens(request.Allergens)
	if err != nil {
		return model.AllergenProfile{}, err
	}
	statementDate, err := parseDate(request.SupplierStatementDate)
	if err != nil {
		return model.AllergenProfile{}, err
	}
	profile := model.AllergenProfile{ProfileCode: code, MaterialName: strings.TrimSpace(request.MaterialName), AllergensJSON: allergens, SourceType: request.SourceType, SupplierStatementDate: statementDate, ProfileStatus: request.ProfileStatus, Version: 1, CreatedBy: actor.ID}
	if err := s.repo.Create(ctx, &profile, AuditScope(actor, requestID)); err != nil {
		return model.AllergenProfile{}, err
	}
	return profile, nil
}

func (s *ProfileService) Get(ctx context.Context, id uint) (map[string]any, error) {
	profile, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.Usage(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"profile": profile, "used_by_routes": usage}, nil
}

func (s *ProfileService) List(ctx context.Context, query dto.ProfileQuery) ([]model.AllergenProfile, int64, error) {
	return s.repo.List(ctx, query)
}

func (s *ProfileService) Update(ctx context.Context, id uint, request dto.UpdateProfileRequest, actor Principal, requestID string) (model.AllergenProfile, error) {
	allergens, err := encodeAllergens(request.Allergens)
	if err != nil {
		return model.AllergenProfile{}, err
	}
	statementDate, err := parseDate(request.SupplierStatementDate)
	if err != nil {
		return model.AllergenProfile{}, err
	}
	profile := model.AllergenProfile{ID: id, MaterialName: strings.TrimSpace(request.MaterialName), AllergensJSON: allergens, SourceType: request.SourceType, SupplierStatementDate: statementDate, ProfileStatus: request.ProfileStatus}
	if err := s.repo.Update(ctx, &profile, request.ExpectedVersion, AuditScope(actor, requestID)); err != nil {
		return model.AllergenProfile{}, err
	}
	return profile, nil
}

func encodeAllergens(items []string) (datatypes.JSON, error) {
	seen := make(map[string]string)
	for _, item := range items {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, exists := seen[key]; !exists {
			seen[key] = clean
		}
	}
	if len(seen) == 0 {
		return nil, NewError(http.StatusBadRequest, "validation_error", "至少需要一个有效过敏原", nil)
	}
	normalized := make([]string, 0, len(seen))
	for _, item := range seen {
		normalized = append(normalized, item)
	}
	sort.Strings(normalized)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode allergens: %w", err)
	}
	return datatypes.JSON(encoded), nil
}

func parseDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, NewError(http.StatusBadRequest, "validation_error", "供应商声明日期格式应为 YYYY-MM-DD", err)
	}
	return &parsed, nil
}

var identifierPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{1,63}$`)

func normalizeIdentifier(value string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if !identifierPattern.MatchString(code) {
		return "", NewError(http.StatusBadRequest, "invalid_code", "代码仅允许字母、数字、点、下划线和连字符，并须以字母或数字开头", nil)
	}
	return code, nil
}
