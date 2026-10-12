package promptdiff

import (
	"strings"
	"testing"
)

// "+2%" hides that a change fixed one case and broke
// a damage claim. The report names both.
func TestReportNamesBrokenCases(t *testing.T) {
	rows := []Row{
		{"claim-0417", "claim", "claim", "other"},     // BROKE
		{"wismo-0102", "wismo", "other", "wismo"},     // FIXED
		{"inv-0033", "invoice", "invoice", "invoice"}, // still right
		{"odd-0007", "claim", "other", "wismo"},       // still wrong
	}
	var b strings.Builder
	report(&b, rows)
	out := b.String()
	for _, want := range []string{
		"BROKE claim-0417: claim -> other",
		"FIXED wismo-0102: other -> wismo",
		"map[BROKE:1 FIXED:1 still right:1 still wrong:1]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Cases where A and B agree are counted, not printed.
	if strings.Contains(out, "inv-0033") || strings.Contains(out, "odd-0007") {
		t.Errorf("unchanged cases printed:\n%s", out)
	}
}
