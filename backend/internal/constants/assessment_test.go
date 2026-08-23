package constants

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name     string
		from, to AssessmentStatus
		want     bool
	}{
		{name: "queue starts", from: AssessmentQueued, to: AssessmentCalculating, want: true},
		{name: "calculation completes", from: AssessmentCalculating, to: AssessmentPendingReview, want: true},
		{name: "calculation retry", from: AssessmentCalculating, to: AssessmentQueued, want: true},
		{name: "review accepts", from: AssessmentPendingReview, to: AssessmentAccepted, want: true},
		{name: "review rejects", from: AssessmentPendingReview, to: AssessmentRejected, want: true},
		{name: "input stales pending", from: AssessmentPendingReview, to: AssessmentStale, want: true},
		{name: "input stales accepted", from: AssessmentAccepted, to: AssessmentStale, want: true},
		{name: "cannot bypass calculation", from: AssessmentQueued, to: AssessmentPendingReview, want: false},
		{name: "cannot repeat review", from: AssessmentAccepted, to: AssessmentRejected, want: false},
		{name: "stale terminal", from: AssessmentStale, to: AssessmentCalculating, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanTransition(test.from, test.to); got != test.want {
				t.Fatalf("CanTransition(%s,%s) = %v, want %v", test.from, test.to, got, test.want)
			}
		})
	}
}
