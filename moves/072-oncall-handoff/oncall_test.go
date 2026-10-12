package oncall

import (
	"testing"
	"time"
)

// Theo silenced disk alerts on a replica. The
// silence ends Tuesday 03:00, inside the next shift.
func TestExpiringSoon(t *testing.T) {
	from := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC) // Monday handoff
	week := 7 * 24 * time.Hour
	ss := []Silence{
		{Alert: "PGDiskHigh replica-2", By: "theo", Ends: time.Date(2026, 3, 10, 3, 0, 0, 0, time.UTC)},
		{Alert: "QuoteLatency", By: "theo", Ends: time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)},
		{Alert: "old", Ends: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},    // already over
		{Alert: "later", Ends: time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)}, // after the shift
	}
	got := ExpiringSoon(ss, from, week)
	if len(got) != 2 || got[0].Alert != "QuoteLatency" || got[1].Alert != "PGDiskHigh replica-2" {
		t.Fatalf("got %+v", got)
	}
}
