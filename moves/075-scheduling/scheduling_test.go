package scheduling

import (
	"testing"
	"time"
)

func zones(t *testing.T, names ...string) []*time.Location {
	var out []*time.Location
	for _, n := range names {
		loc, err := time.LoadLocation(n)
		if err != nil {
			t.Skip("no tzdata:", err)
		}
		out = append(out, loc)
	}
	return out
}

// The book's Try it: New York and Los Angeles, 9 to 17.
// No slot may start before 9:00 in Los Angeles.
func TestNoSlotBeforeNineInLA(t *testing.T) {
	locs := zones(t, "America/New_York", "America/Los_Angeles")
	la := locs[1]
	r := Rules{Minutes: 30, Earliest: 9, Latest: 17}
	from := time.Date(2026, 3, 9, 8, 0, 0, 0, locs[0]) // Monday 8:00 New York
	got := Slots(from, r, locs, 50, func(time.Time, time.Time) bool { return false })
	if len(got) == 0 {
		t.Fatal("no slots")
	}
	for _, s := range got {
		if s.In(la).Hour() < 9 {
			t.Fatalf("slot %v starts before 9:00 in LA", s.In(la))
		}
		if e := s.Add(30 * time.Minute).In(locs[0]); e.Hour()*60+e.Minute() > 17*60 {
			t.Fatalf("slot ends after 17:00 in NY: %v", e)
		}
	}
	// First valid start: 9:00 LA = 12:00 NY.
	if f := got[0].In(la); f.Hour() != 9 || f.Minute() != 0 {
		t.Fatalf("first slot = %v", f)
	}
}

// Dana's 8:00 Monday in Atlanta was 5:00 in Seattle.
func TestAtlantaEightIsSeattleFive(t *testing.T) {
	locs := zones(t, "America/New_York", "America/Los_Angeles")
	s := time.Date(2026, 3, 9, 8, 0, 0, 0, locs[0])
	if inHours(s, s.Add(30*time.Minute), Rules{Earliest: 9, Latest: 17}, locs) {
		t.Fatal("8:00 Eastern must fail for a Pacific attendee")
	}
}

// Using time.Location means a DST change never moves
// a slot: 9:00 local before and after the switch.
func TestDaylightSaving(t *testing.T) {
	locs := zones(t, "America/New_York")
	r := Rules{Earliest: 9, Latest: 17}
	before := time.Date(2026, 3, 6, 9, 0, 0, 0, locs[0]) // Fri, EST
	after := time.Date(2026, 3, 9, 9, 0, 0, 0, locs[0])  // Mon, EDT
	for _, s := range []time.Time{before, after} {
		if !inHours(s, s.Add(time.Hour), r, locs) {
			t.Errorf("9:00 local rejected on %v", s)
		}
	}
	sat := time.Date(2026, 3, 7, 10, 0, 0, 0, locs[0])
	if inHours(sat, sat.Add(time.Hour), r, locs) {
		t.Error("Saturday accepted")
	}
}
