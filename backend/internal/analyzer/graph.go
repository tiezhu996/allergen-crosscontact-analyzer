package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/dto"
	"food-allergen-crosscontact-analyzer/backend/internal/model"
)

type Node struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ProfileID uint   `json:"profile_id"`
	Order     int    `json:"order"`
}

type Edge struct {
	ID                   uint    `json:"id"`
	From                 string  `json:"from"`
	To                   string  `json:"to"`
	ContactType          string  `json:"contact_type"`
	SharedEquipment      string  `json:"shared_equipment"`
	CleaningFactor       float64 `json:"cleaning_factor"`
	CarryoverProbability float64 `json:"carryover_probability"`
	Weight               float64 `json:"weight"`
	EvidenceNote         string  `json:"evidence_note"`
	Version              uint    `json:"version"`
}

type Graph struct {
	Nodes    map[string]Node   `json:"nodes"`
	Outgoing map[string][]Edge `json:"outgoing"`
	Edges    []Edge            `json:"edges"`
}

func BuildGraph(steps []dto.RouteStep, records []model.ContactEdge) (Graph, error) {
	graph := Graph{Nodes: make(map[string]Node), Outgoing: make(map[string][]Edge)}
	if len(steps) < 2 {
		return Graph{}, fmt.Errorf("route graph requires at least two steps")
	}
	for index, step := range steps {
		code := strings.TrimSpace(step.StepCode)
		if code == "" {
			return Graph{}, fmt.Errorf("step at position %d has no code", index+1)
		}
		if _, exists := graph.Nodes[code]; exists {
			return Graph{}, fmt.Errorf("duplicate step code %q", code)
		}
		graph.Nodes[code] = Node{Code: code, Name: strings.TrimSpace(step.StepName), ProfileID: step.ProfileID, Order: index}
	}
	seen := make(map[string]struct{})
	for _, record := range records {
		if !record.Enabled {
			continue
		}
		if _, ok := graph.Nodes[record.FromStepCode]; !ok {
			return Graph{}, fmt.Errorf("contact edge %d references unknown source step %q", record.ID, record.FromStepCode)
		}
		if _, ok := graph.Nodes[record.ToStepCode]; !ok {
			return Graph{}, fmt.Errorf("contact edge %d references unknown target step %q", record.ID, record.ToStepCode)
		}
		if record.FromStepCode == record.ToStepCode {
			return Graph{}, fmt.Errorf("contact edge %d cannot be a self-loop", record.ID)
		}
		key := fmt.Sprintf("%s\x00%s", record.FromStepCode, record.ToStepCode)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		weight, err := EdgeWeight(record.CleaningFactor, record.CarryoverProbability)
		if err != nil {
			return Graph{}, fmt.Errorf("contact edge %d: %w", record.ID, err)
		}
		edge := Edge{
			ID: record.ID, From: record.FromStepCode, To: record.ToStepCode,
			ContactType: record.ContactType, SharedEquipment: record.SharedEquipment,
			CleaningFactor: record.CleaningFactor, CarryoverProbability: record.CarryoverProbability,
			Weight: weight, EvidenceNote: record.EvidenceNote, Version: record.Version,
		}
		graph.Edges = append(graph.Edges, edge)
		graph.Outgoing[edge.From] = append(graph.Outgoing[edge.From], edge)
	}
	for code := range graph.Outgoing {
		sort.Slice(graph.Outgoing[code], func(i, j int) bool {
			left, right := graph.Outgoing[code][i], graph.Outgoing[code][j]
			if left.To == right.To {
				return left.ID < right.ID
			}
			return left.To < right.To
		})
	}
	sort.Slice(graph.Edges, func(i, j int) bool { return graph.Edges[i].ID < graph.Edges[j].ID })
	return graph, nil
}

func DetectCycles(graph Graph) [][]string {
	cycles := make([][]string, 0)
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	stack := make([]string, 0, len(graph.Nodes))
	var visit func(string)
	visit = func(code string) {
		visiting[code] = true
		stack = append(stack, code)
		for _, edge := range graph.Outgoing[code] {
			if visiting[edge.To] {
				start := 0
				for index, item := range stack {
					if item == edge.To {
						start = index
						break
					}
				}
				cycle := stack[start:]
				cycle = append(cycle, edge.To)
				cycles = append(cycles, cycle)
			} else if !visited[edge.To] {
				visit(edge.To)
			}
		}
		stack = stack[:len(stack)-1]
		visiting[code] = false
		visited[code] = true
	}
	codes := make([]string, 0, len(graph.Nodes))
	for code := range graph.Nodes {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if !visited[code] {
			visit(code)
		}
	}
	return cycles
}
