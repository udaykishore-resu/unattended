package postmortem

import (
	"slices"
	"testing"
)

func TestUnsourced(t *testing.T) {
	facts := `quote_p99_before_ms=410 quote_p99_worst_ms=4200
errors_per_min_worst=212 loads_unassigned=1,340 window=14:02-14:51`
	draft := "Quote p99 rose from 410 ms to 4200 ms. Errors peaked at " +
		"212 per minute and 1,340 loads went unassigned. The rollback " +
		"took 15 minutes. About 18% of quotes failed."
	got := Unsourced(draft, facts)
	if !slices.Equal(got, []string{"15", "18%"}) {
		t.Fatalf("unsourced = %q", got)
	}
}
