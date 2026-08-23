package analyzer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"food-allergen-crosscontact-analyzer/backend/internal/constants"
)

type ProfileSeed struct {
	ProfileID    uint     `json:"profile_id"`
	ProfileCode  string   `json:"profile_code"`
	MaterialName string   `json:"material_name"`
	Version      uint     `json:"version"`
	Allergens    []string `json:"allergens"`
}

type EdgeEvidence struct {
	EdgeID               uint    `json:"edge_id"`
	From                 string  `json:"from"`
	To                   string  `json:"to"`
	ContactType          string  `json:"contact_type"`
	SharedEquipment      string  `json:"shared_equipment"`
	CleaningFactor       float64 `json:"cleaning_factor"`
	CarryoverProbability float64 `json:"carryover_probability"`
	EdgeWeight           float64 `json:"edge_weight"`
	EvidenceNote         string  `json:"evidence_note"`
	EdgeVersion          uint    `json:"edge_version"`
}

type RiskItem struct {
	Allergen          string              `json:"allergen"`
	SourceProfileID   uint                `json:"source_profile_id"`
	SourceProfileCode string              `json:"source_profile_code"`
	SourceMaterial    string              `json:"source_material"`
	SourceStepCode    string              `json:"source_step_code"`
	TargetStepCode    string              `json:"target_step_code"`
	TargetStepName    string              `json:"target_step_name"`
	Path              []string            `json:"path"`
	RawScore          float64             `json:"raw_score"`
	RiskLevel         constants.RiskLevel `json:"risk_level"`
	Declared          bool                `json:"declared"`
	CriticalEdge      *EdgeEvidence       `json:"critical_edge"`
	CleaningEvidence  []EdgeEvidence      `json:"cleaning_evidence"`
	ThresholdVersion  string              `json:"threshold_version"`
}

type MatrixCell struct {
	TargetStepCode string              `json:"target_step_code"`
	TargetStepName string              `json:"target_step_name"`
	Allergen       string              `json:"allergen"`
	MaxRawScore    float64             `json:"max_raw_score"`
	RiskLevel      constants.RiskLevel `json:"risk_level"`
	PathCount      int                 `json:"path_count"`
	Declared       bool                `json:"declared"`
}

type Result struct {
	RiskItems           []RiskItem          `json:"risk_items"`
	Matrix              []MatrixCell        `json:"matrix"`
	DeclaredAllergens   []string            `json:"declared_allergens"`
	PropagatedAllergens []string            `json:"propagated_allergens"`
	HighestRiskLevel    constants.RiskLevel `json:"highest_risk_level"`
	Thresholds          ThresholdSnapshot   `json:"thresholds"`
	Cycles              [][]string          `json:"cycles"`
	CycleEdgesSkipped   int                 `json:"cycle_edges_skipped"`
	DepthLimitReached   int                 `json:"depth_limit_reached"`
	MaxDepth            int                 `json:"max_depth"`
}

type pathState struct {
	current string
	path    []string
	edges   []Edge
	score   float64
	visited map[string]bool
}

func Propagate(graph Graph, profiles map[uint]ProfileSeed, declared []string, maxDepth int, thresholds ThresholdSnapshot) (Result, error) {
	return propagate(context.Background(), graph, profiles, declared, maxDepth, thresholds)
}

// PropagateCtx runs the cross-contact propagation and aborts early when ctx is
// cancelled or its deadline expires. Callers that drive propagation from an HTTP
// request should prefer this over Propagate so client cancellation propagates
// into the (potentially long-running) graph traversal.
func PropagateCtx(ctx context.Context, graph Graph, profiles map[uint]ProfileSeed, declared []string, maxDepth int, thresholds ThresholdSnapshot) (Result, error) {
	return propagate(ctx, graph, profiles, declared, maxDepth, thresholds)
}

func propagate(ctx context.Context, graph Graph, profiles map[uint]ProfileSeed, declared []string, maxDepth int, thresholds ThresholdSnapshot) (Result, error) {
	if maxDepth < 1 {
		return Result{}, fmt.Errorf("max propagation depth must be positive")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	declaredSet := normalizedSet(declared)
	result := Result{DeclaredAllergens: sortedKeys(declaredSet), Thresholds: thresholds, Cycles: DetectCycles(graph), MaxDepth: maxDepth}
	codes := make([]string, 0, len(graph.Nodes))
	for code := range graph.Nodes {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return graph.Nodes[codes[i]].Order < graph.Nodes[codes[j]].Order })
	for _, sourceCode := range codes {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		node := graph.Nodes[sourceCode]
		profile, ok := profiles[node.ProfileID]
		if !ok {
			return Result{}, fmt.Errorf("step %q references unavailable allergen profile %d", sourceCode, node.ProfileID)
		}
		for _, allergen := range normalizeAllergens(profile.Allergens) {
			walkSource(ctx, graph, sourceCode, allergen, profile, declaredSet, maxDepth, thresholds, &result)
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
		}
	}
	result.Matrix = buildMatrix(result.RiskItems, thresholds)
	propagated := make(map[string]struct{})
	for _, item := range result.RiskItems {
		propagated[item.Allergen] = struct{}{}
	}
	result.PropagatedAllergens = sortedKeys(propagated)
	result.HighestRiskLevel = HighestRisk(result.RiskItems)
	sort.Slice(result.RiskItems, func(i, j int) bool {
		left, right := result.RiskItems[i], result.RiskItems[j]
		if left.RawScore != right.RawScore {
			return left.RawScore > right.RawScore
		}
		if left.Allergen != right.Allergen {
			return left.Allergen < right.Allergen
		}
		if left.TargetStepCode != right.TargetStepCode {
			return left.TargetStepCode < right.TargetStepCode
		}
		return strings.Join(left.Path, "\x00") < strings.Join(right.Path, "\x00")
	})
	return result, nil
}

func walkSource(ctx context.Context, graph Graph, source, allergen string, profile ProfileSeed, declared map[string]struct{}, maxDepth int, thresholds ThresholdSnapshot, result *Result) {
	initial := pathState{current: source, path: []string{source}, score: 1, visited: map[string]bool{source: true}}
	stack := []pathState{initial}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return
		}
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(state.edges) >= maxDepth {
			if len(graph.Outgoing[state.current]) > 0 {
				result.DepthLimitReached++
			}
			continue
		}
		outgoing := graph.Outgoing[state.current]
		for index := len(outgoing) - 1; index >= 0; index-- {
			edge := outgoing[index]
			if state.visited[edge.To] {
				result.CycleEdgesSkipped++
				continue
			}
			score := AccumulateScore(state.score, edge.Weight)
			if score <= 0 {
				continue
			}
			path := appendCopy(state.path, edge.To)
			edges := appendEdge(state.edges, edge)
			evidence := evidenceFor(edges)
			critical := CriticalEdge(edges)
			var criticalEvidence *EdgeEvidence
			if critical != nil {
				item := evidenceFromEdge(*critical)
				criticalEvidence = &item
			}
			_, isDeclared := declared[strings.ToLower(allergen)]
			target := graph.Nodes[edge.To]
			result.RiskItems = append(result.RiskItems, RiskItem{
				Allergen: allergen, SourceProfileID: profile.ProfileID, SourceProfileCode: profile.ProfileCode,
				SourceMaterial: profile.MaterialName, SourceStepCode: source, TargetStepCode: target.Code,
				TargetStepName: target.Name, Path: path, RawScore: score, RiskLevel: thresholds.Map(score),
				Declared: isDeclared, CriticalEdge: criticalEvidence, CleaningEvidence: evidence,
				ThresholdVersion: thresholds.Version,
			})
			visited := cloneVisited(state.visited)
			visited[edge.To] = true
			stack = append(stack, pathState{current: edge.To, path: path, edges: edges, score: score, visited: visited})
		}
	}
}

func buildMatrix(items []RiskItem, thresholds ThresholdSnapshot) []MatrixCell {
	type key struct{ step, allergen string }
	cells := make(map[key]MatrixCell)
	for _, item := range items {
		k := key{step: item.TargetStepCode, allergen: item.Allergen}
		cell, exists := cells[k]
		if !exists {
			cell = MatrixCell{TargetStepCode: item.TargetStepCode, TargetStepName: item.TargetStepName, Allergen: item.Allergen, Declared: item.Declared}
		}
		cell.PathCount++
		if item.RawScore > cell.MaxRawScore {
			cell.MaxRawScore = item.RawScore
			cell.RiskLevel = thresholds.Map(item.RawScore)
		}
		cells[k] = cell
	}
	result := make([]MatrixCell, 0, len(cells))
	for _, cell := range cells {
		result = append(result, cell)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TargetStepCode == result[j].TargetStepCode {
			return result[i].Allergen < result[j].Allergen
		}
		return result[i].TargetStepCode < result[j].TargetStepCode
	})
	return result
}

func evidenceFor(edges []Edge) []EdgeEvidence {
	result := make([]EdgeEvidence, len(edges))
	for i, edge := range edges {
		result[i] = evidenceFromEdge(edge)
	}
	return result
}
func evidenceFromEdge(edge Edge) EdgeEvidence {
	return EdgeEvidence{EdgeID: edge.ID, From: edge.From, To: edge.To, ContactType: edge.ContactType, SharedEquipment: edge.SharedEquipment, CleaningFactor: edge.CleaningFactor, CarryoverProbability: edge.CarryoverProbability, EdgeWeight: edge.Weight, EvidenceNote: edge.EvidenceNote, EdgeVersion: edge.Version}
}
func cloneVisited(source map[string]bool) map[string]bool {
	target := make(map[string]bool, len(source)+1)
	for key, value := range source {
		target[key] = value
	}
	return target
}
func appendCopy(source []string, value string) []string {
	target := make([]string, len(source), len(source)+1)
	copy(target, source)
	return append(target, value)
}
func appendEdge(source []Edge, value Edge) []Edge {
	target := make([]Edge, len(source), len(source)+1)
	copy(target, source)
	return append(target, value)
}

func normalizeAllergens(items []string) []string {
	seen := make(map[string]string)
	for _, item := range items {
		clean := strings.TrimSpace(item)
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
func normalizedSet(items []string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, item := range normalizeAllergens(items) {
		result[strings.ToLower(item)] = struct{}{}
	}
	return result
}
func sortedKeys(items map[string]struct{}) []string {
	result := make([]string, 0, len(items))
	for key := range items {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
