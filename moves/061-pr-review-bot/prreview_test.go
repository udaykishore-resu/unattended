package prreview

import "testing"

func TestSilenceIsTheDefault(t *testing.T) {
	diff := Added{"quote/rate.go": {
		41: "\treturn q.Lane.Rate * miles",
		42: "\trows, _ := db.Query(sql)",
	}}
	kinds := map[string]bool{"nil-deref": true, "unchecked-error": true,
		"sql-injection": true, "data-race": true, "resource-leak": true}
	fs := []Finding{
		// The real bug: kept.
		{File: "quote/rate.go", Line: 41, Kind: "nil-deref",
			Evidence: "q.Lane.Rate", Confidence: "high"},
		// Misread diff: quote not on that line.
		{File: "quote/rate.go", Line: 42, Kind: "nil-deref",
			Evidence: "q.Lane.Rate", Confidence: "high"},
		// Style is the linter's job.
		{File: "quote/rate.go", Line: 41, Kind: "naming",
			Evidence: "miles", Confidence: "high"},
		// Not sure enough.
		{File: "quote/rate.go", Line: 42, Kind: "unchecked-error",
			Evidence: "rows, _ :=", Confidence: "medium"},
		// Not an added line.
		{File: "quote/old.go", Line: 7, Kind: "data-race",
			Evidence: "x++", Confidence: "high"},
		// No evidence at all.
		{File: "quote/rate.go", Line: 42, Kind: "resource-leak",
			Evidence: "", Confidence: "high"},
	}
	got := Keep(fs, diff, kinds)
	if len(got) != 1 || got[0].Line != 41 || got[0].Kind != "nil-deref" {
		t.Fatalf("kept %+v", got)
	}
}
