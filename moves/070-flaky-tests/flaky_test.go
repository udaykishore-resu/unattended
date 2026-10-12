package flaky

import (
	"maps"
	"testing"
)

func TestFlaky(t *testing.T) {
	runs := []Run{
		// Same code, two outcomes, on two commits.
		{"TestAssignLoad_Concurrent", "a1", true},
		{"TestAssignLoad_Concurrent", "a1", false},
		{"TestAssignLoad_Concurrent", "b2", false},
		{"TestAssignLoad_Concurrent", "b2", true},
		// Broken on one commit, fixed on the next: not flaky.
		{"TestQuoteTotal", "a1", false},
		{"TestQuoteTotal", "b2", true},
		// Always green.
		{"TestHealthz", "a1", true},
		{"TestHealthz", "a1", true},
	}
	want := map[string]int{"TestAssignLoad_Concurrent": 2}
	if got := Flaky(runs); !maps.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
}
