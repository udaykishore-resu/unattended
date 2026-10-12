package searcheval

import (
	"math"
	"testing"
)

func TestScore(t *testing.T) {
	results := map[string][]string{
		"brightline det rate atl??": {"a", "b", "c", "det-rates"}, // rank 4
		"edi down":                  {"edi-runbook"},              // rank 1
		"no answer here":            {"x", "y"},                   // miss
	}
	cases := []Case{
		{"brightline det rate atl??", []string{"det-rates"}},
		{"edi down", []string{"edi-runbook", "edi-old"}},
		{"no answer here", []string{"z"}},
	}
	search := func(q string) []string { return results[q] }

	rec5, mrr := Score(5, cases, search)
	if math.Abs(rec5-2.0/3) > 1e-9 {
		t.Errorf("recall@5 = %v", rec5)
	}
	// Rank 4 scores 0.25, rank 1 scores 1.
	if math.Abs(mrr-(1.25/3)) > 1e-9 {
		t.Errorf("MRR = %v", mrr)
	}
	rec3, _ := Score(3, cases, search)
	if math.Abs(rec3-1.0/3) > 1e-9 {
		t.Errorf("recall@3 = %v", rec3)
	}
}
