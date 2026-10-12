package searcheval

import (
	"math"
	"testing"
)

// Several wanted docs in the top k count the query
// once: recall stays at or below 1.
func TestManyWantedDocsCountOnce(t *testing.T) {
	cases := []Case{{"edi down", []string{"a", "b", "c"}}}
	search := func(string) []string { return []string{"a", "b", "c"} }
	rec, mrr, err := Score(5, cases, search)
	if err != nil || rec != 1 || mrr != 1 {
		t.Fatalf("rec=%v mrr=%v err=%v; want 1, 1", rec, mrr, err)
	}
}

func TestEmptySetIsAnError(t *testing.T) {
	rec, mrr, err := Score(5, nil, func(string) []string { return nil })
	if err == nil || math.IsNaN(rec) || math.IsNaN(mrr) {
		t.Fatalf("rec=%v mrr=%v err=%v", rec, mrr, err)
	}
}

// Duplicate IDs: k counts result slots, the chunks
// the model would see. A wanted doc counts once, at
// its first slot; an unwanted doc repeated by two
// chunks takes two slots.
func TestDuplicateIDs(t *testing.T) {
	cases := []Case{{"q", []string{"want"}}}
	dupWanted := func(string) []string {
		return []string{"want", "want", "want"}
	}
	rec, mrr, _ := Score(1, cases, dupWanted)
	if rec != 1 || mrr != 1 {
		t.Fatalf("dup wanted: rec=%v mrr=%v", rec, mrr)
	}
	dupOther := func(string) []string {
		return []string{"x", "x", "want"}
	}
	rec, mrr, _ = Score(2, cases, dupOther)
	if rec != 0 || math.Abs(mrr-1.0/3) > 1e-9 {
		t.Fatalf("dup other: rec=%v mrr=%v", rec, mrr)
	}
}
