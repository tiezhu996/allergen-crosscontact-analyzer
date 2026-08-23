package analyzer

import (
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
)

func TestHighestRiskCriticalWins(t *testing.T) {
	items := []RiskItem{
		{Allergen: "Peanut", RawScore: 0.4, RiskLevel: constants.RiskHigh},
		{Allergen: "Soy", RawScore: 0.7, RiskLevel: constants.RiskCritical},
	}
	if got := HighestRisk(items); got != constants.RiskCritical {
		t.Fatalf("highest = %q, want critical (critical must outrank high)", got)
	}
}

func TestHighestRiskMediumBeatsLow(t *testing.T) {
	items := []RiskItem{
		{Allergen: "Peanut", RawScore: 0.1, RiskLevel: constants.RiskLow},
		{Allergen: "Soy", RawScore: 0.2, RiskLevel: constants.RiskMedium},
	}
	if got := HighestRisk(items); got != constants.RiskMedium {
		t.Fatalf("highest = %q, want medium (medium must outrank low)", got)
	}
}

func TestMapBoundaryAtThreshold(t *testing.T) {
	thresholds := ThresholdSnapshot{Medium: 0.12, High: 0.35, Critical: 0.65, Version: "2026.1"}
	if got := thresholds.Map(0.65); got != constants.RiskCritical {
		t.Fatalf("score == critical threshold mapped to %q, want critical", got)
	}
	if got := thresholds.Map(0.35); got != constants.RiskHigh {
		t.Fatalf("score == high threshold mapped to %q, want high", got)
	}
	if got := thresholds.Map(0.12); got != constants.RiskMedium {
		t.Fatalf("score == medium threshold mapped to %q, want medium", got)
	}
}

func TestHighestRiskEmptyIsLow(t *testing.T) {
	if got := HighestRisk(nil); got != constants.RiskLow {
		t.Fatalf("highest of empty items = %q, want low", got)
	}
}
