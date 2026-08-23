package analyzer

import (
	"testing"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
)

func TestBuildGraphValidation(t *testing.T) {
	base := []dto.RouteStep{{StepCode: "A", StepName: "A step", ProfileID: 1}, {StepCode: "B", StepName: "B step", ProfileID: 2}}
	tests := []struct {
		name    string
		steps   []dto.RouteStep
		edges   []model.ContactEdge
		wantErr bool
	}{
		{name: "valid", steps: base, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", CleaningFactor: .5, CarryoverProbability: .8, Enabled: true}}},
		{name: "duplicate step", steps: append(base, base[0]), wantErr: true},
		{name: "unknown endpoint", steps: base, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "C", Enabled: true}}, wantErr: true},
		{name: "self loop", steps: base, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "A", Enabled: true}}, wantErr: true},
		{name: "disabled invalid ignored", steps: base, edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "C", Enabled: false}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildGraph(test.steps, test.edges)
			if (err != nil) != test.wantErr {
				t.Fatalf("BuildGraph() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name  string
		edges []model.ContactEdge
		want  int
	}{
		{name: "acyclic", edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", Enabled: true}}},
		{name: "one cycle", edges: []model.ContactEdge{{ID: 1, FromStepCode: "A", ToStepCode: "B", Enabled: true}, {ID: 2, FromStepCode: "B", ToStepCode: "A", Enabled: true}}, want: 1},
	}
	steps := []dto.RouteStep{{StepCode: "A", StepName: "A step", ProfileID: 1}, {StepCode: "B", StepName: "B step", ProfileID: 2}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph, err := BuildGraph(steps, test.edges)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(DetectCycles(graph)); got != test.want {
				t.Fatalf("DetectCycles() = %d, want %d", got, test.want)
			}
		})
	}
}
