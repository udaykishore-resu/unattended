package status

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"
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

// The limit, pinned: an allowed number on the wrong
// metric passes. The book says so.
func TestStrayIsNarrow(t *testing.T) {
	f := Facts{Week: 11, MergedPRs: ptr(31), Incidents: ptr(2)}
	if got := Stray("We merged 2 PRs and had 31 incidents.", f.Allowed()); len(got) != 0 {
		t.Fatalf("Stray got smarter (%v); update the book", got)
	}
}

// A down source is nil, not zero, so "0 deploys"
// can't sneak through as a fact.
func TestDownSourceIsNotZero(t *testing.T) {
	f := Facts{Week: 11, MergedPRs: ptr(31)}
	if got := Stray("0 deploys this week", f.Allowed()); !slices.Equal(got, []int{0}) {
		t.Fatalf("got %v", got)
	}
	if Show("deploys", f.Deploys) != "deploys: not available" {
		t.Fatal(Show("deploys", f.Deploys))
	}
}

func TestReleaseBlocksAndDatesAllowed(t *testing.T) {
	loc := time.UTC
	f := Facts{Week: 41, MergedPRs: ptr(31),
		From: time.Date(2026, 10, 5, 0, 0, 0, 0, loc),
		To:   time.Date(2026, 10, 12, 0, 0, 0, 0, loc)}
	if err := Release("Week 41 (Oct 5 to Oct 11, 2026): 31 PRs merged.", f); err != nil {
		t.Fatal(err)
	}
	var se ErrStray
	if err := Release("Week 41: 34 PRs merged.", f); !errors.As(err, &se) || !slices.Equal(se.Numbers, []int{34}) {
		t.Fatalf("got %v", err)
	}
}

// A PR merged at 23:30 Sunday belongs to the week
// that ends that night in the team's zone, even
// though it is already Monday in UTC.
func TestWeekBoundaryInTeamZone(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, ny) // Friday
	from, to := Week(now, ny)
	if !from.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, ny)) || !to.Equal(time.Date(2026, 10, 19, 0, 0, 0, 0, ny)) {
		t.Fatalf("range %v - %v", from, to)
	}
	merged := time.Date(2026, 10, 18, 23, 30, 0, 0, ny) // Sunday night
	if merged.UTC().Weekday() != time.Monday {
		t.Fatal("test setup: should be Monday in UTC")
	}
	if merged.Before(from) || !merged.Before(to) {
		t.Fatal("Sunday 23:30 local fell outside its week")
	}
	// Across the DST change (Sun Nov 1, 2026) the week is still Mon-Mon local: 169 hours.
	from, to = Week(time.Date(2026, 10, 28, 9, 0, 0, 0, ny), ny)
	if to.Sub(from) != 7*24*time.Hour+time.Hour || from.Hour() != 0 || to.Hour() != 0 {
		t.Fatalf("DST week %v - %v", from, to)
	}
}

func TestCountPaginatesDedupesAndFailsClosed(t *testing.T) {
	pages := map[string]Page{
		"":   {IDs: []string{"pr1", "pr2"}, Next: "p2"},
		"p2": {IDs: []string{"pr2", "pr3"}, Next: "p3"}, // pr2 repeated
		"p3": {IDs: []string{"pr4"}},
	}
	n, np := Count(func(c string) (Page, error) { return pages[c], nil })
	if n == nil || *n != 4 || np != 3 {
		t.Fatalf("n=%v pages=%d", n, np)
	}
	n, _ = Count(func(c string) (Page, error) {
		if c == "p3" {
			return Page{}, errors.New("502")
		}
		return pages[c], nil
	})
	if n != nil {
		t.Fatalf("partial failure gave a count %d", *n)
	}
}

// Stored facts round-trip, nil stays nil, so a rerun
// against a new prompt sees the same "not available".
func TestFactsJSONRoundTrip(t *testing.T) {
	f := Facts{Week: 41, MergedPRs: ptr(31), SlippedIDs: []string{"OPS-12"}}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var g Facts
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	if g.Deploys != nil || *g.MergedPRs != 31 || g.Week != 41 {
		t.Fatalf("got %+v from %s", g, b)
	}
}
