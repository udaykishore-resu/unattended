package flaky

import (
	"maps"
	"testing"
)

func TestFlaky(t *testing.T) {
	runs := []Run{
		// Same code, two outcomes, on two commits.
		{Test: "TestAssignLoad_Concurrent", SHA: "a1", Passed: true},
		{Test: "TestAssignLoad_Concurrent", SHA: "a1", Passed: false},
		{Test: "TestAssignLoad_Concurrent", SHA: "b2", Passed: false},
		{Test: "TestAssignLoad_Concurrent", SHA: "b2", Passed: true},
		// Broken on one commit, fixed on the next: not flaky.
		{Test: "TestQuoteTotal", SHA: "a1", Passed: false},
		{Test: "TestQuoteTotal", SHA: "b2", Passed: true},
		// Always green.
		{Test: "TestHealthz", SHA: "a1", Passed: true},
		{Test: "TestHealthz", SHA: "a1", Passed: true},
	}
	want := map[string]int{"TestAssignLoad_Concurrent": 2}
	if got := Flaky(runs); !maps.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
}
