package status

import (
	"slices"
	"testing"
)

func ptr(n int) *int { return &n }

// The draft said 34 merged PRs; the real count was 31.
func TestStrayCatchesTheThirtyFour(t *testing.T) {
	f := Facts{Week: 11, MergedPRs: ptr(31), Incidents: ptr(2)}
	draft := "Week 11: we merged 34 PRs across dispatch and quoting, " +
		"and handled 2 incidents."
	if got := Stray(draft, f.Allowed()); !slices.Equal(got, []int{34}) {
		t.Fatalf("got %v", got)
	}
	ok := "Week 11: we merged 31 PRs and handled 2 incidents. Deploys: not available."
	if got := Stray(ok, f.Allowed()); len(got) != 0 {
		t.Fatalf("clean draft flagged: %v", got)
	}
}

// A down source is nil, not zero, so "0 deploys"
// can't sneak through as a fact.
func TestDownSourceIsNotZero(t *testing.T) {
	f := Facts{Week: 11, MergedPRs: ptr(31)}
	if got := Stray("0 deploys this week", f.Allowed()); !slices.Equal(got, []int{0}) {
		t.Fatalf("got %v", got)
	}
}
