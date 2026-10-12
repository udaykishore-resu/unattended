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
		{"odd-0007", "claim", "other", "wismo"},       // still wrong, moved
		{"odd-0008", "claim", "other", "other"},       // still wrong, same
	}
	var b strings.Builder
	report(&b, rows)
	out := b.String()
	for _, want := range []string{
		"BROKE claim-0417: claim -> other",
		"FIXED wismo-0102: other -> wismo",
		"still wrong odd-0007: other -> wismo",
		"map[BROKE:1 FIXED:1 still right:1 still wrong:2]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Cases where A and B give the same answer are
	// counted, not printed.
	if strings.Contains(out, "inv-0033") || strings.Contains(out, "odd-0008") {
		t.Errorf("unchanged cases printed:\n%s", out)
	}
}

// One table of known transitions, one row each, with
// the bucket and whether the line prints.
func TestBuckets(t *testing.T) {
	cases := []struct {
		name   string
		row    Row
		bucket string
		prints bool
	}{
		{"right to wrong", Row{"c1", "claim", "claim", "other"}, "BROKE", true},
		{"wrong to right", Row{"c2", "claim", "other", "claim"}, "FIXED", true},
		{"wrong to other wrong", Row{"c3", "claim", "other", "wismo"}, "still wrong", true},
		{"wrong to same wrong", Row{"c4", "claim", "other", "other"}, "still wrong", false},
		{"right to right", Row{"c5", "claim", "claim", "claim"}, "still right", false},
	}
	for _, c := range cases {
		var b strings.Builder
		report(&b, []Row{c.row})
		out := b.String()
		if !strings.Contains(out, "map["+c.bucket+":1]") {
			t.Errorf("%s: bucket line %q", c.name, out)
		}
		if got := strings.Contains(out, c.row.ID); got != c.prints {
			t.Errorf("%s: printed=%v, want %v:\n%s", c.name, got, c.prints, out)
		}
	}
}
