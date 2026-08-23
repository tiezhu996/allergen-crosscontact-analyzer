package analyzer

import (
	"fmt"
	"math"
)

func EdgeWeight(cleaningFactor, carryoverProbability float64) (float64, error) {
	if cleaningFactor < 0 || cleaningFactor > 1 {
		return 0, fmt.Errorf("cleaning factor %.4f is outside [0,1]", cleaningFactor)
	}
	if carryoverProbability < 0 || carryoverProbability > 1 {
		return 0, fmt.Errorf("carryover probability %.4f is outside [0,1]", carryoverProbability)
	}
	return roundScore((1 - cleaningFactor) * carryoverProbability), nil
}

func AccumulateScore(current, edgeWeight float64) float64 {
	if current <= 0 || edgeWeight <= 0 {
		return 0
	}
	return roundScore(current * edgeWeight)
}

func roundScore(value float64) float64 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return math.Round(value*1_000_000) / 1_000_000
}

func CriticalEdge(edges []Edge) *Edge {
	if len(edges) == 0 {
		return nil
	}
	critical := edges[0]
	for _, edge := range edges[1:] {
		if edge.Weight > critical.Weight || (edge.Weight == critical.Weight && edge.ID < critical.ID) {
			critical = edge
		}
	}
	return &critical
}
