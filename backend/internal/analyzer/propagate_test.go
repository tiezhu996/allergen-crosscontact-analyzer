package analyzer

import (
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/config"
	"food-allergen-crosscontact-analyzer/backend/internal/constants"
	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
)

func TestPropagate(t *testing.T) {
	thresholds, err := NewThresholdSnapshot(config.Thresholds{Medium: .1, High: .3, Critical: .6, Version: "test-v1"})
	if err != nil {
		t.Fatal(err)
	}
	steps := []dto.RouteStep{{StepCode: "A", StepName: "Source", ProfileID: 1}, {StepCode: "B", StepName: "Middle", ProfileID: 2}, {StepCode: "C", StepName: "Target", ProfileID: 2}}
	profiles := map[uint]ProfileSeed{1: {ProfileID: 1, ProfileCode: "P-1", MaterialName: "Peanut paste", Version: 1, Allergens: []string{"Peanut"}}, 2: {ProfileID: 2, ProfileCode: "P-2", MaterialName: "Milk powder", Version: 1, Allergens: []string{"Milk"}}}
	tests := []struct {
		name                     string
		edges                    []model.ContactEdge
		maxDepth                 int
		wantPeanutScore          float64
		wantCycleSkip, wantDepth int
	}{
		{name: "weighted two edge path", maxDepth: 4, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", CleaningFactor: .5, CarryoverProbability: .8, EvidenceNote: "edge one", Enabled: true}, {ID: 2, FromStepCode: "B", ToStepCode: "C", CleaningFactor: .25, CarryoverProbability: .5, EvidenceNote: "edge two", Enabled: true}}, wantPeanutScore: .15},
		{name: "cycle terminates", maxDepth: 5, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", CleaningFactor: 0, CarryoverProbability: .5, Enabled: true}, {ID: 2, FromStepCode: "B", ToStepCode: "A", CleaningFactor: 0, CarryoverProbability: .5, Enabled: true}}, wantPeanutScore: .5, wantCycleSkip: 1},
		{name: "depth limit", maxDepth: 1, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", CleaningFactor: 0, CarryoverProbability: .5, Enabled: true}, {ID: 2, FromStepCode: "B", ToStepCode: "C", CleaningFactor: 0, CarryoverProbability: .5, Enabled: true}}, wantPeanutScore: .5, wantDepth: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph, err := BuildGraph(steps, test.edges)
			if err != nil {
				t.Fatal(err)
			}
			result, err := Propagate(graph, profiles, []string{"Milk"}, test.maxDepth, thresholds)
			if err != nil {
				t.Fatal(err)
			}
			var lowest float64 = 2
			for _, item := range result.RiskItems {
				if item.Allergen == "Peanut" && item.RawScore < lowest {
					lowest = item.RawScore
				}
				if item.Allergen == "Peanut" && item.Declared {
					t.Fatal("propagated peanut must remain separate from declared milk")
				}
			}
			if lowest != test.wantPeanutScore {
				t.Fatalf("lowest peanut score = %v, want %v", lowest, test.wantPeanutScore)
			}
			if result.CycleEdgesSkipped < test.wantCycleSkip {
				t.Fatalf("cycle skips = %d, want at least %d", result.CycleEdgesSkipped, test.wantCycleSkip)
			}
			if result.DepthLimitReached < test.wantDepth {
				t.Fatalf("depth limit count = %d, want at least %d", result.DepthLimitReached, test.wantDepth)
			}
		})
	}
}

func TestThresholdMap(t *testing.T) {
	thresholds := ThresholdSnapshot{Medium: .1, High: .3, Critical: .6, Version: "v1"}
	tests := []struct {
		score float64
		want  constants.RiskLevel
	}{{.05, constants.RiskLow}, {.1, constants.RiskMedium}, {.3, constants.RiskHigh}, {.6, constants.RiskCritical}}
	for _, test := range tests {
		if got := thresholds.Map(test.score); got != test.want {
			t.Errorf("Map(%v) = %s, want %s", test.score, got, test.want)
		}
	}
}
