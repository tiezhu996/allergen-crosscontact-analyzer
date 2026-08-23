package analyzer

import "testing"

func TestEdgeWeight(t *testing.T) {
	tests := []struct {
		name                      string
		cleaning, carryover, want float64
		wantErr                   bool
	}{
		{name: "no cleaning", cleaning: 0, carryover: .8, want: .8},
		{name: "partial cleaning", cleaning: .25, carryover: .8, want: .6},
		{name: "complete cleaning", cleaning: 1, carryover: .8, want: 0},
		{name: "invalid cleaning", cleaning: 1.2, carryover: .5, wantErr: true},
		{name: "invalid carryover", cleaning: .2, carryover: -.1, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := EdgeWeight(test.cleaning, test.carryover)
			if (err != nil) != test.wantErr {
				t.Fatalf("EdgeWeight() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("EdgeWeight() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAccumulateScore(t *testing.T) {
	tests := []struct {
		name                string
		current, edge, want float64
	}{
		{name: "single path", current: 1, edge: .45, want: .45},
		{name: "two stage", current: .45, edge: .435, want: .19575},
		{name: "zero edge", current: .8, edge: 0, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AccumulateScore(test.current, test.edge); got != test.want {
				t.Fatalf("AccumulateScore() = %v, want %v", got, test.want)
			}
		})
	}
}
