package oncall

import (
	"testing"
	"time"
)

// A silence that ends exactly at a handoff appears in
// exactly one note: the shift it ends in. With
// Before(until) it fell between two notes and
// appeared in neither.
func TestBoundaryInExactlyOneNote(t *testing.T) {
	mon := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)
	week := 7 * 24 * time.Hour
	next := mon.Add(week)
	ss := []Silence{
		{Alert: "at-next-handoff", Ends: next},
		{Alert: "at-this-handoff", Ends: mon},
		{Alert: "one-ns-after", Ends: next.Add(time.Nanosecond)},
	}
	count := map[string]int{}
	for _, from := range []time.Time{mon.Add(-week), mon, next} {
		for _, s := range ExpiringSoon(ss, from, week) {
			count[s.Alert]++
		}
	}
	for _, a := range []string{"at-next-handoff", "at-this-handoff", "one-ns-after"} {
		if count[a] != 1 {
			t.Errorf("%s listed %d times", a, count[a])
		}
	}
	// This week's note includes the one ending at the
	// next handoff, but not the one ending now.
	got := ExpiringSoon(ss, mon, week)
	if len(got) != 1 || got[0].Alert != "at-next-handoff" {
		t.Fatalf("got %+v", got)
	}
}

// An open-ended silence (zero end time) is never
// "expiring"; the book's fix lists old silences too.
func TestOpenEndedNotListed(t *testing.T) {
	mon := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)
	if got := ExpiringSoon([]Silence{{Alert: "forever"}}, mon, 7*24*time.Hour); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}
