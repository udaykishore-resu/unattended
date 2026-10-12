package postmortem

import (
	"slices"
	"testing"
)

// Each case pins what the detector does with one
// number format. "flag" means the reviewer sees it.
func TestNumberFormats(t *testing.T) {
	facts := `quote_p99_worst_ms=4200 loads_unassigned=1340
error_rate_pct=7.8% p99_before_ms=410 change_pct=-12%
date=2026-03-09 window=14:02-14:51`
	cases := []struct {
		draft string
		flag  []string
	}{
		{"1,340 loads went unassigned.", nil},             // separator normalized
		{"Errors reached 7.8% at peak.", nil},             // percentage, same form
		{"Errors reached 7.8 per cent.", []string{"7.8"}}, // % missing: flag
		{"p99 peaked at 4.2 seconds.", []string{"4.2"}},   // unit change: flag
		{"p99 sat at 41 ms before.", []string{"41"}},      // substring of 410: flag
		{"Quotes fell 12% (-12%).", nil},                  // sign is not checked
		{"On 2026-03-09 from 14:02 to 14:51.", nil},       // dates split into parts
		{"On 2026-03-10 at 14:02.", []string{"10"}},       // wrong day: flag
		{"Rollback took 15 minutes.", []string{"15"}},     // from nowhere: flag
		{"Step 2 of the runbook.", nil},                   // single digits skipped
	}
	for _, c := range cases {
		if got := Unsourced(c.draft, facts); !slices.Equal(got, c.flag) {
			t.Errorf("%q: flagged %q, want %q", c.draft, got, c.flag)
		}
	}
}

// A known limit: a match does not show the number
// describes the same metric. 410 is p99 before, not
// errors per minute, and still passes.
func TestMatchIsNotProvenance(t *testing.T) {
	facts := "p99_before_ms=410 errors_per_min=212"
	if got := Unsourced("Errors peaked at 410 per minute.", facts); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}
