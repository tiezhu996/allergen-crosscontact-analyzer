package analyzer

import (
	"fmt"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
)

type ThresholdSnapshot struct {
	Medium   float64 `json:"medium"`
	High     float64 `json:"high"`
	Critical float64 `json:"critical"`
	Version  string  `json:"version"`
}

func NewThresholdSnapshot(source config.Thresholds) (ThresholdSnapshot, error) {
	result := ThresholdSnapshot{Medium: source.Medium, High: source.High, Critical: source.Critical, Version: source.Version}
	if result.Version == "" {
		return ThresholdSnapshot{}, fmt.Errorf("threshold version is required")
	}
	if !(0 < result.Medium && result.Medium < result.High && result.High < result.Critical && result.Critical <= 1) {
		return ThresholdSnapshot{}, fmt.Errorf("thresholds must be ascending values in (0,1]")
	}
	return result, nil
}

func (t ThresholdSnapshot) Map(score float64) constants.RiskLevel {
	switch {
	case score > t.Critical:
		return constants.RiskCritical
	case score > t.High:
		return constants.RiskHigh
	case score > t.Medium:
		return constants.RiskMedium
	default:
		return constants.RiskLow
	}
}

func HighestRisk(items []RiskItem) constants.RiskLevel {
	var highest constants.RiskLevel
	for _, item := range items {
		if constants.RiskRank(item.RiskLevel) > constants.RiskRank(highest) {
			highest = item.RiskLevel
		}
	}
	return highest
}
