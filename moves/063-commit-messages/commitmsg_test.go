package commitmsg

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

const why = "carrier API times out under load at month end"

func TestTrimDropsGeneratedAndNamesCuts(t *testing.T) {
	big := strings.Repeat("+x\n", 3000)
	kept, cut := Trim([]FileDiff{
		{"go.sum", "+h1:abc\n"},
		{"api/carrier.pb.go", big},
		{"dispatch/carrier.go", big},
		{"dispatch/retry.go", "+retry once\n"},
	}, MaxPatch)
	if len(kept) != 2 || kept[0].Path != "dispatch/carrier.go" ||
		len(kept[0].Patch) != MaxPatch {
		t.Fatalf("kept %+v", kept)
	}
	if len(cut) != 1 || cut[0] != "dispatch/carrier.go" {
		t.Fatalf("cut %v", cut)
	}
}

// The cap can't hide a change silently: the model is
// told the view is partial, and code appends a
// trailer the model can't remove.
func TestTruncationIsNeverSilent(t *testing.T) {
	m := &llm.FakeModel{Responses: []llm.Response{{
		Text: "Retry carrier lookup once on timeout\n\nThe carrier API times out at month end."}}}
	fs := []FileDiff{{"dispatch/carrier.go", strings.Repeat("+x\n", 3000)}}
	msg, ok, err := Draft(context.Background(), m, why, fs)
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if !strings.HasSuffix(msg, "Diff-Truncated: dispatch/carrier.go") {
		t.Fatalf("no trailer:\n%s", msg)
	}
	in := m.Calls[0].Messages[0].Content
	if !strings.Contains(in, "Truncated diffs (partial view): dispatch/carrier.go") ||
		!strings.Contains(in, why) {
		t.Fatalf("prompt:\n%s", in)
	}
	if m.Calls[0].System != System || !m.Calls[0].CacheSystem {
		t.Fatal("system prompt must be the fixed, cached one")
	}
}

func TestRefusesEmptyWhy(t *testing.T) {
	m := &llm.FakeModel{}
	if _, _, err := Draft(context.Background(), m, "  ", nil); !errors.Is(err, ErrNoWhy) {
		t.Fatal(err)
	}
	if len(m.Calls) != 0 {
		t.Fatal("model called without a why line")
	}
}

type slow struct{}

func (slow) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	<-ctx.Done()
	return llm.Response{}, ctx.Err()
}

// Slow or down model: the editor opens with only the
// why line, within the timeout.
func TestFallsBackToWhyLine(t *testing.T) {
	old := Timeout
	Timeout = 50 * time.Millisecond
	defer func() { Timeout = old }()
	start := time.Now()
	msg, ok, err := Draft(context.Background(), slow{}, why, nil)
	if err != nil || ok || msg != why {
		t.Fatalf("msg=%q ok=%v err=%v", msg, ok, err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("hook hung for %v", d)
	}
	down := &llm.FakeModel{Err: errors.New("503")}
	if msg, ok, _ := Draft(context.Background(), down, why, nil); ok || msg != why {
		t.Fatalf("down: %q %v", msg, ok)
	}
}
