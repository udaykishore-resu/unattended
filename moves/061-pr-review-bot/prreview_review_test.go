package prreview

import (
	"context"
	"errors"
	"testing"
)

var kinds = map[string]bool{"nil-deref": true, "unchecked-error": true,
	"sql-injection": true, "data-race": true, "resource-leak": true}

// Keep checks that the quote is present on that
// added line. It does not check that the finding is
// right: a correct quote with a wrong claim passes.
func TestEvidenceIsPresenceNotCorrectness(t *testing.T) {
	diff := Added{"quote/rate.go": {41: "\treturn q.Lane.Rate * miles"}}
	wrong := Finding{File: "quote/rate.go", Line: 41, Kind: "data-race",
		Evidence: "q.Lane.Rate", Confidence: "high"}
	if got := Keep([]Finding{wrong}, diff, kinds); len(got) != 1 {
		t.Fatalf("presence check should pass a quoted but wrong finding: %+v", got)
	}
}

// A refactor that moves code: every moved line
// counts as added, so Keep alone would keep a
// finding on it. DropMoved removes pure moves.
func TestMovedCodeIsNotNewCode(t *testing.T) {
	moved := "\trows, _ := db.Query(sql)"
	diff := Added{"quote/store.go": {
		10: moved,
		11: "\tdefer rows.Close()", // new line
	}}
	f := Finding{File: "quote/store.go", Line: 10, Kind: "unchecked-error",
		Evidence: "rows, _ :=", Confidence: "high"}
	if got := Keep([]Finding{f}, diff, kinds); len(got) != 1 {
		t.Fatalf("without move detection the finding is kept: %+v", got)
	}
	d := DropMoved(diff, []string{"  rows, _ := db.Query(sql)  "})
	if got := Keep([]Finding{f}, d, kinds); len(got) != 0 {
		t.Fatalf("moved line still reviewed: %+v", got)
	}
	if _, ok := d["quote/store.go"][11]; !ok {
		t.Fatal("new line dropped")
	}
}

func findings() []Finding {
	return []Finding{
		{File: "a.go", Line: 1, Kind: "unchecked-error", Evidence: "x", Confidence: "high"},
		{File: "a.go", Line: 2, Kind: "nil-deref", Evidence: "y", Confidence: "high"},
		{File: "a.go", Line: 3, Kind: "resource-leak", Evidence: "z", Confidence: "high"},
		{File: "a.go", Line: 4, Kind: "data-race", Evidence: "w", Confidence: "high"},
	}
}

var order = []string{"nil-deref", "data-race", "sql-injection",
	"resource-leak", "unchecked-error"}

// The author pushes (or amends) while the bot runs:
// the head SHA changes before posting, so the run is
// dropped and nothing lands on the wrong lines.
func TestHeadMovedRunDropped(t *testing.T) {
	sent := 0
	head := func(context.Context) (string, error) { return "b2c3d4e", nil }
	send := func(context.Context, Review) error { sent++; return nil }
	err := Post(context.Background(), "a1b2c3d", findings(), order, 3, head, send)
	if !errors.Is(err, ErrStale) || sent != 0 {
		t.Fatalf("err=%v sent=%d", err, sent)
	}
}

// Same head: one review, pinned to the reviewed SHA,
// at most three findings, worst kind first.
func TestPinnedToReviewedSHA(t *testing.T) {
	var got Review
	head := func(context.Context) (string, error) { return "a1b2c3d", nil }
	send := func(_ context.Context, r Review) error { got = r; return nil }
	if err := Post(context.Background(), "a1b2c3d", findings(), order, 3, head, send); err != nil {
		t.Fatal(err)
	}
	if got.CommitID != "a1b2c3d" || len(got.Findings) != 3 {
		t.Fatalf("got %+v", got)
	}
	want := []string{"nil-deref", "data-race", "resource-leak"}
	for i, k := range want {
		if got.Findings[i].Kind != k {
			t.Fatalf("order %d = %s, want %s", i, got.Findings[i].Kind, k)
		}
	}
}

// No findings: no call to the code host at all.
func TestSilenceMakesNoCall(t *testing.T) {
	head := func(context.Context) (string, error) { t.Fatal("head called"); return "", nil }
	send := func(context.Context, Review) error { t.Fatal("send called"); return nil }
	if err := Post(context.Background(), "a1b2c3d", nil, order, 3, head, send); err != nil {
		t.Fatal(err)
	}
}
