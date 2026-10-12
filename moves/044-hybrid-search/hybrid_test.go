package hybrid

import (
	"reflect"
	"slices"
	"testing"
)

// The book's worked example: the PRO ticket is first
// on keyword and absent on vector (~0.016); the
// runbook is fifth and first (~0.032), so it wins,
// but the ticket now sits near the top.
func TestPROTicketSurfaces(t *testing.T) {
	keyword := []string{"ticket-PRO-4471", "kw2", "kw3", "kw4", "runbook-appts"}
	vector := []string{"runbook-appts", "v2", "v3", "v4", "v5"}
	got := Fuse(60, keyword, vector)
	if got[0] != "runbook-appts" {
		t.Fatalf("top = %s", got[0])
	}
	if i := slices.Index(got, "ticket-PRO-4471"); i != 1 {
		t.Fatalf("PRO ticket at %d, want 1 (was rank 31 on vectors alone)", i)
	}
}

func TestTieBreakIsStable(t *testing.T) {
	a := Fuse(60, []string{"b"}, []string{"a"})
	if !reflect.DeepEqual(a, []string{"a", "b"}) {
		t.Fatalf("got %v", a)
	}
	for range 20 {
		if b := Fuse(60, []string{"b"}, []string{"a"}); !reflect.DeepEqual(a, b) {
			t.Fatal("order changed between runs")
		}
	}
}
