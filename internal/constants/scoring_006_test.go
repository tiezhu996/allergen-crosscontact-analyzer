package constants

import "testing"

func TestCanTransitionAllowsCalculationRollback(t *testing.T) {
	if !CanTransition(AssessmentCalculating, AssessmentQueued) {
		t.Fatal("calculating -> queued must be a legal transition so failed runs can roll back")
	}
}
