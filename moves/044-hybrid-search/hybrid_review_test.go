package hybrid

import (
	"reflect"
	"strings"
	"testing"
)

// An ID repeated inside one list counts once, at its
// best rank: two chunks of one doc can't double it.
func TestDuplicateWithinListCountsOnce(t *testing.T) {
	got := Fuse(60, []string{"a", "a", "b"}, []string{"b"})
	// Counted once, a is 1/61 and b is 1/63 + 1/61.
	// Counting the repeat would give a 1/61 + 1/62
	// and put it first.
	if !reflect.DeepEqual(got, []string{"b", "a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestEmptyLists(t *testing.T) {
	if got := Fuse(60); len(got) != 0 {
		t.Fatalf("no lists: %v", got)
	}
	if got := Fuse(60, nil, []string{}); len(got) != 0 {
		t.Fatalf("empty lists: %v", got)
	}
	// One side empty (a timed-out search): the other
	// side's order is kept.
	got := Fuse(60, nil, []string{"z", "y", "x"})
	if !reflect.DeepEqual(got, []string{"z", "y", "x"}) {
		t.Fatalf("one empty side: %v", got)
	}
}

// Rank 1 on one list and rank 1 on another tie
// exactly; the ID breaks the tie, every run.
func TestTiesAcrossLists(t *testing.T) {
	for range 50 {
		got := Fuse(60, []string{"q", "r"}, []string{"p", "s"})
		want := []string{"p", "q", "r", "s"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// The first result has rank 1: its score is 1/(k+1).
// With k=0, x scores 1, y scores 1/2 + 1/2 and z
// scores 2; x and y tie and the ID breaks it. Ranks
// from 0 would give x and z +Inf and put x first.
func TestRankStartsAtOne(t *testing.T) {
	got := Fuse(0, []string{"x"}, []string{"z", "y"},
		[]string{"z", "y"})
	if !reflect.DeepEqual(got, []string{"z", "x", "y"}) {
		t.Fatalf("got %v", got)
	}
}

// Both queries filter tenant and expiry before the
// ORDER BY and LIMIT. The integration test runs them.
func TestFiltersSitBeforeLimit(t *testing.T) {
	for name, q := range map[string]string{
		"vector": VectorSQL, "keyword": KeywordSQL} {
		order := strings.Index(q, "ORDER BY")
		for _, p := range []string{"tenant_id = $1",
			"expires_at IS NULL OR expires_at >= now()"} {
			i := strings.Index(q, p)
			if i < 0 || i > order {
				t.Errorf("%s: %q not in WHERE before ORDER BY", name, p)
			}
		}
		if !strings.HasSuffix(q, "LIMIT 50") {
			t.Errorf("%s: LIMIT must come last", name)
		}
	}
}
