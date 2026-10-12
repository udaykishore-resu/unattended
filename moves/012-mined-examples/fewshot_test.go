package fewshot

import "testing"

// 140 of 212 corrections were "where is my truck"
// emails filed as claims. "Newest ten" would be all
// of those; a per-label cap spreads the picks.
func TestCapSpreadsAcrossLabels(t *testing.T) {
	var log []Fix
	for range 140 {
		log = append(log, Fix{Input: "where is my truck", Got: "claim", Want: "wismo"})
	}
	log = append(log,
		Fix{Input: "inv 12 wrong", Got: "other", Want: "invoice_dispute"},
		Fix{Input: "inv 13 wrong", Got: "other", Want: "invoice_dispute"},
		Fix{Input: "inv 14 wrong", Got: "other", Want: "invoice_dispute"},
		Fix{Input: "crate crushed", Got: "other", Want: "damage_claim"},
	)
	got := pickShots(log, 2, 10)
	count := map[string]int{}
	for _, f := range got {
		count[f.Want]++
	}
	if count["wismo"] != 2 || count["invoice_dispute"] != 2 || count["damage_claim"] != 1 {
		t.Fatalf("per-label counts = %v", count)
	}
}

func TestLimit(t *testing.T) {
	log := []Fix{{Want: "a"}, {Want: "b"}, {Want: "c"}, {Want: "d"}}
	if got := pickShots(log, 2, 3); len(got) != 3 {
		t.Fatalf("got %d shots, want 3", len(got))
	}
}
