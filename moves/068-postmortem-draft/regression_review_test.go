package postmortem

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	enrich "github.com/udaykishore-resu/unattended/moves/066-alert-enrichment"
)

// Part VII regression case, from the synthesis: the
// enricher may say "possibly related", and nothing
// downstream may upgrade that to "caused by" or
// "fixed it" until the review confirms a factor.

type src struct{ note string }

func (s src) Runbook(context.Context, string) (string, error) { return "step 2", nil }
func (s src) Deploys(context.Context, string, time.Time) ([]enrich.Deploy, error) {
	return []enrich.Deploy{{Service: "ratequote", Ref: "region-filter"}}, nil
}
func (s src) Note(context.Context, enrich.Enriched) (string, error) { return s.note, nil }

func TestPossiblyRelatedIsNeverUpgraded(t *testing.T) {
	a := enrich.Alert{Name: "QuoteErrors", Service: "ratequote",
		FiredAt: time.Date(2026, 3, 9, 16, 2, 0, 0, time.UTC)}

	// 1. The page: "likely cause" is dropped in code;
	// "possibly related" goes out with the facts.
	if e := enrich.Enrich(context.Background(), src{"region-filter deploy: likely cause"}, a); e.Note != "" {
		t.Fatalf("causal note reached the page: %q", e.Note)
	}
	if e := enrich.Enrich(context.Background(), src{"region-filter deploy 18 min ago, possibly related"}, a); e.Note == "" {
		t.Fatal("tentative note dropped")
	}

	// 2. The draft does not start from an unsigned
	// timeline.
	in := Inputs{Facts: "error_rate_pct=7.8% recovered=16:09"}
	if _, err := Check(in, "", ""); !errors.Is(err, ErrUnsigned) {
		t.Fatalf("unsigned timeline accepted: %v", err)
	}

	// 3. The model's summary upgrades the note. Before
	// review confirms anything, both claims are flagged.
	in.TimelineSignedBy = "theo"
	summary := "The region-filter deploy caused the errors. " +
		"Rolling back region-filter fixed it. " +
		"Errors reached 7.8%."
	f, err := Check(in, summary, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"The region-filter deploy caused the errors.",
		"Rolling back region-filter fixed it."}
	if !slices.Equal(f.Causes, want) {
		t.Fatalf("causes flagged %q", f.Causes)
	}

	// 4. The tentative wording passes untouched.
	f, _ = Check(in, "The region-filter deploy was possibly related.", "")
	if len(f.Causes) != 0 {
		t.Fatalf("tentative sentence flagged: %q", f.Causes)
	}

	// 5. After review confirms the real factor, a causal
	// sentence that names it passes; the old claim
	// still does not.
	in.Confirmed = []string{"carrier API outage"}
	f, _ = Check(in, "A carrier API outage caused the errors. "+
		"The rollback fixed it.", "")
	if !slices.Equal(f.Causes, []string{"The rollback fixed it."}) {
		t.Fatalf("after review: %q", f.Causes)
	}
}
