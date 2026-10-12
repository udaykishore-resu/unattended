package digest

import (
	"testing"
	"time"
)

func TestPick(t *testing.T) {
	now := time.Date(2026, 10, 7, 6, 0, 0, 0, time.UTC)
	ago := func(d int) time.Time { return now.Add(-time.Duration(d)*24*time.Hour - time.Hour) }
	notes := []Note{
		{"retry-budgets.md", ago(90), ago(80)},                   // 90 days: back
		{"carrier-timeouts.md", ago(3), now.Add(-2 * time.Hour)}, // fresh
		{"week-old.md", ago(7), ago(7)},                          // 7 days: back
		{"random.md", ago(12), ago(12)},                          // not on a spacing day
	}
	fresh, old := Pick(notes, now)
	if len(fresh) != 1 || fresh[0].Path != "carrier-timeouts.md" {
		t.Fatalf("fresh = %+v", fresh)
	}
	if len(old) != 2 || old[0].Path != "retry-budgets.md" || old[1].Path != "week-old.md" {
		t.Fatalf("old = %+v", old)
	}
}
