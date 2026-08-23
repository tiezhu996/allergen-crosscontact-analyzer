package analyzer

import (
	"reflect"
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
)

func chainGraph() (Graph, map[uint]ProfileSeed, []string, ThresholdSnapshot) {
	steps := []dto.RouteStep{
		{StepCode: "A-01", StepName: "A", ProfileID: 1},
		{StepCode: "B-02", StepName: "B", ProfileID: 1},
		{StepCode: "C-03", StepName: "C", ProfileID: 1},
		{StepCode: "D1-04", StepName: "D1", ProfileID: 1},
		{StepCode: "D2-05", StepName: "D2", ProfileID: 1},
	}
	records := []model.ContactEdge{
		{ID: 1, RouteID: 1, FromStepCode: "A-01", ToStepCode: "B-02", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 2, RouteID: 1, FromStepCode: "B-02", ToStepCode: "C-03", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 3, RouteID: 1, FromStepCode: "C-03", ToStepCode: "D1-04", ContactType: "shared_line", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 4, RouteID: 1, FromStepCode: "C-03", ToStepCode: "D2-05", ContactType: "shared_line", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
	}
	graph, err := BuildGraph(steps, records)
	if err != nil {
		panic(err)
	}
	seeds := map[uint]ProfileSeed{1: {ProfileID: 1, ProfileCode: "MAT-1", MaterialName: "m1", Version: 1, Allergens: []string{"Peanut"}}}
	thresholds := ThresholdSnapshot{Medium: 0.05, High: 0.3, Critical: 0.6, Version: "2026.1"}
	return graph, seeds, []string{}, thresholds
}

// Two sibling expansions must not overwrite each other's path tail.
func TestRiskItemPathsNotCorrupted(t *testing.T) {
	graph, seeds, declared, thresholds := chainGraph()
	result, err := Propagate(graph, seeds, declared, 6, thresholds)
	if err != nil {
		t.Fatalf("propagate: %v", err)
	}
	paths := map[string][]string{}
	for _, item := range result.RiskItems {
		paths[item.TargetStepCode] = item.Path
	}
	wantD1 := []string{"A-01", "B-02", "C-03", "D1-04"}
	wantD2 := []string{"A-01", "B-02", "C-03", "D2-05"}
	if !reflect.DeepEqual(paths["D1-04"], wantD1) {
		t.Fatalf("path to D1 = %v, want %v (sibling expansion overwrote it)", paths["D1-04"], wantD1)
	}
	if !reflect.DeepEqual(paths["D2-05"], wantD2) {
		t.Fatalf("path to D2 = %v, want %v", paths["D2-05"], wantD2)
	}
}

// Cycle paths must not be corrupted by later DFS stack pops.
func TestDetectCyclesPathsNotCorrupted(t *testing.T) {
	steps := []dto.RouteStep{
		{StepCode: "A-01", StepName: "A", ProfileID: 1},
		{StepCode: "B-02", StepName: "B", ProfileID: 1},
		{StepCode: "C-03", StepName: "C", ProfileID: 1},
		{StepCode: "D-04", StepName: "D", ProfileID: 1},
	}
	records := []model.ContactEdge{
		{ID: 1, RouteID: 1, FromStepCode: "A-01", ToStepCode: "B-02", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 2, RouteID: 1, FromStepCode: "B-02", ToStepCode: "C-03", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 3, RouteID: 1, FromStepCode: "C-03", ToStepCode: "A-01", ContactType: "rework", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 4, RouteID: 1, FromStepCode: "C-03", ToStepCode: "D-04", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
	}
	graph, err := BuildGraph(steps, records)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	cycles := DetectCycles(graph)
	if len(cycles) != 1 {
		t.Fatalf("cycles = %v, want exactly one", cycles)
	}
	want := []string{"A-01", "B-02", "C-03", "A-01"}
	if !reflect.DeepEqual(cycles[0], want) {
		t.Fatalf("cycle = %v, want %v (later DFS stack pops corrupted it)", cycles[0], want)
	}
}

// Already-visited nodes must not be re-entered while walking a graph with a cycle.
func TestCycleNotRevisited(t *testing.T) {
	steps := []dto.RouteStep{
		{StepCode: "A-01", StepName: "A", ProfileID: 1},
		{StepCode: "B-02", StepName: "B", ProfileID: 1},
	}
	records := []model.ContactEdge{
		{ID: 1, RouteID: 1, FromStepCode: "A-01", ToStepCode: "B-02", ContactType: "sequence", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
		{ID: 2, RouteID: 1, FromStepCode: "B-02", ToStepCode: "A-01", ContactType: "rework", SharedEquipment: "eq", CleaningFactor: 0.5, CarryoverProbability: 0.9, EvidenceNote: "note", Enabled: true, Version: 1},
	}
	graph, err := BuildGraph(steps, records)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	seeds := map[uint]ProfileSeed{1: {ProfileID: 1, ProfileCode: "MAT-1", MaterialName: "m1", Version: 1, Allergens: []string{"Peanut"}}}
	thresholds := ThresholdSnapshot{Medium: 0.05, High: 0.3, Critical: 0.6, Version: "2026.1"}
	result, err := Propagate(graph, seeds, []string{}, 6, thresholds)
	if err != nil {
		t.Fatalf("propagate: %v", err)
	}
	for _, item := range result.RiskItems {
		for _, code := range item.Path {
			if code != item.SourceStepCode && code != item.TargetStepCode && countOccurrences(item.Path, code) > 1 {
				t.Fatalf("path %v re-visits node %q; visited nodes must be skipped", item.Path, code)
			}
		}
	}
	if result.CycleEdgesSkipped == 0 {
		t.Fatalf("cycle edges should be skipped during propagation")
	}
}

func countOccurrences(items []string, target string) int {
	n := 0
	for _, item := range items {
		if item == target {
			n++
		}
	}
	return n
}

// Equal-weight edges must pick the smaller edge id as the critical edge.
func TestCriticalEdgePrefersSmallerID(t *testing.T) {
	edges := []Edge{
		{ID: 9, From: "A-01", To: "B-02", Weight: 0.5},
		{ID: 3, From: "B-02", To: "C-03", Weight: 0.5},
	}
	critical := CriticalEdge(edges)
	if critical == nil {
		t.Fatal("critical edge is nil")
	}
	if critical.ID != 3 {
		t.Fatalf("critical edge id = %d, want 3 (equal weights must prefer the smaller id)", critical.ID)
	}
}
