package enrich

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// hung blocks every source until its context ends,
// the way a stuck wiki, CD API or model would.
type hung struct{ noteBudget time.Duration }

func (h *hung) Runbook(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (h *hung) Deploys(ctx context.Context, _ string, _ time.Time) ([]Deploy, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (h *hung) Note(ctx context.Context, _ Enriched) (string, error) {
	if dl, ok := ctx.Deadline(); ok {
		h.noteBudget = time.Until(dl)
	}
	<-ctx.Done()
	return "", ctx.Err()
}

// Every source hangs: Enrich still returns the alert,
// names what is missing, and leaves the note empty.
// The caller's deadline stands in for the 8 s cap.
func TestEverySourceDownPageStillGoesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	a := Alert{"QuoteLatencyHigh", "ratequote", fired}
	start := time.Now()
	e := Enrich(ctx, &hung{}, a)
	if time.Since(start) > 2*time.Second {
		t.Fatal("enrichment held the page")
	}
	if e.Alert != a || e.Note != "" ||
		!slices.Equal(e.Missing, []string{"runbook", "deploys"}) {
		t.Fatalf("got %+v", e)
	}
}

// The model's budget is its own: at most 3 s, inside
// whatever is left of the 8 s cap.
func TestNoteHasItsOwnShorterTimeout(t *testing.T) {
	h := &hung{}
	ns := &noteOnly{hung: h}
	start := time.Now()
	e := Enrich(context.Background(), ns, Alert{Name: "QuoteLatencyHigh", FiredAt: fired})
	took := time.Since(start)
	if h.noteBudget <= 0 || h.noteBudget > 3*time.Second {
		t.Fatalf("note budget %v", h.noteBudget)
	}
	if took > 4*time.Second || e.Runbook == "" || e.Note != "" {
		t.Fatalf("took %v, got %+v", took, e)
	}
}

type noteOnly struct{ *hung }

func (noteOnly) Runbook(context.Context, string) (string, error) {
	return "Step 2: check DB pool", nil
}

func (noteOnly) Deploys(context.Context, string, time.Time) ([]Deploy, error) {
	return nil, nil
}

type noter struct {
	src
	note string
	err  error
}

func (n *noter) Note(context.Context, Enriched) (string, error) { return n.note, n.err }

// "Possibly related" passes; a note that claims a
// cause or a fix is dropped, and the facts go out.
func TestCausalNoteDropped(t *testing.T) {
	cases := map[string]bool{
		"deploy changed pool size, possibly related": true,
		"check DB pool first":                        true,
		"likely cause: region-filter deploy":         false,
		"Root cause is the pool size change":         false,
		"rolling back region-filter fixed it":        false,
		"errors due to the deploy":                   false,
		"deploy caused the latency":                  false,
	}
	for note, kept := range cases {
		n := &noter{note: note}
		e := Enrich(context.Background(), n, Alert{Name: "QuoteLatencyHigh", FiredAt: fired})
		if (e.Note != "") != kept || e.Runbook == "" {
			t.Errorf("%q: note=%q", note, e.Note)
		}
	}
	n := &noter{err: errors.New("model 503")}
	if e := Enrich(context.Background(), n, Alert{Name: "x", FiredAt: fired}); e.Note != "" || e.Runbook == "" {
		t.Fatalf("model down: %+v", e)
	}
}
