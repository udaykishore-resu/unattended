package fewshot

import "testing"

// 140 of 212 corrections were "where is my truck"
// emails filed as claims. "Newest ten" would be all
// of those; a per-label cap spreads the picks.
func TestCapSpreadsAcrossLabels(t *testing.T) {
	var log []Fix
	for i := range 140 {
		log = append(log, Fix{Input: "where is my truck " + string(rune('a'+i%26)) + string(rune('a'+i/26)), Got: "claim", Want: "wismo"})
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
	log := []Fix{{Input: "1", Want: "a"}, {Input: "2", Want: "b"},
		{Input: "3", Want: "c"}, {Input: "4", Want: "d"}}
	if got := pickShots(log, 2, 3); len(got) != 3 {
		t.Fatalf("got %d shots, want 3", len(got))
	}
}

// A forwarded email corrected twice (or the same text
// sent by two shippers) must not fill a label's slots
// twice.
func TestDuplicateInputsPickedOnce(t *testing.T) {
	log := []Fix{
		{Input: "where is HF-1", Got: "claim", Want: "wismo"},
		{Input: "where is HF-1", Got: "other", Want: "wismo"},
		{Input: "where is HF-2", Got: "claim", Want: "wismo"},
	}
	got := pickShots(log, 2, 10)
	if len(got) != 2 || got[0].Input == got[1].Input {
		t.Fatalf("picks = %+v", got)
	}
}

// Held out by date: nothing in the eval set was
// available to become an example.
func TestSplitKeepsEvalHeldOut(t *testing.T) {
	rows := []Row{
		{Fix{Input: "old-1", Want: "wismo"}, 3},
		{Fix{Input: "old-2", Want: "claim"}, 20},
		{Fix{Input: "new-1", Want: "claim"}, 36},
		{Fix{Input: "new-2", Want: "wismo"}, 41},
	}
	dev, eval := split(rows, 35)
	shots := pickShots(dev, 2, 10)
	held := map[string]bool{}
	for _, f := range eval {
		held[f.Input] = true
	}
	for _, s := range shots {
		if held[s.Input] {
			t.Fatalf("%s is both an example and an eval case", s.Input)
		}
	}
	if len(shots) != 2 || len(eval) != 2 {
		t.Fatalf("shots %d eval %d", len(shots), len(eval))
	}
}
