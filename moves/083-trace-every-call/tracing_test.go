package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	budget "github.com/udaykishore-resu/unattended/moves/089-token-budgets"
)

// adapter stands in for a vendor adapter: it times
// out once, then answers.
type adapter struct{ calls int }

func (a *adapter) Complete(context.Context, llm.Request) (llm.Response, error) {
	a.calls++
	if a.calls == 1 {
		return llm.Response{}, errors.New("timeout")
	}
	return llm.Response{Text: "reply", StopReason: llm.StopEnd, Usage: llm.Usage{
		InputTokens: 120, CacheReadTokens: 6000, CacheWriteTokens: 0, OutputTokens: 40}}, nil
}

func lines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

// The book's one order, outside in: retry, budget,
// tracer, vendor adapter.
func TestWrapperOrder(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	ledger := budget.NewLedger(map[string]int{"ops-triage": 1_000_000})
	ad := &adapter{}
	m := Retrying{Base: time.Millisecond, Ceil: time.Millisecond,
		Next: Budgeted{
			Ledger: ledger,
			Key:    func(ctx context.Context) string { return From(ctx).Workflow },
			EstIn:  budget.EstBytes,
			Next:   Traced{Next: ad, Model: "small", Log: log}}}

	ctx := With(context.Background(), Info{Workflow: "ops-triage",
		Version: "3f9a1c2b7d4e", TraceID: "t-1", CallID: "c-1"})
	if _, err := m.Complete(ctx, llm.Request{System: "sys", MaxTokens: 100}); err != nil {
		t.Fatal(err)
	}

	got := lines(t, &buf)
	if len(got) != 2 {
		t.Fatalf("want one line per attempt, got %d", len(got))
	}
	for i, l := range got {
		if l["attempt"] != float64(i+1) || l["prompt_version"] != "3f9a1c2b7d4e" ||
			l["model"] != "small" || l["workflow"] != "ops-triage" {
			t.Errorf("line %d = %v", i, l)
		}
	}
	// Failed calls get a record too.
	if got[0]["status"] != "error" || got[1]["status"] != "ok" ||
		got[1]["cache_read"] != float64(6000) || got[1]["stop"] != "end" {
		t.Fatalf("lines = %v", got)
	}
	// The budget sits inside the retry: both attempts were charged.
	if used := ledger.Used("ops-triage"); used <= 6160 {
		t.Fatalf("used = %d; the failed attempt was not charged", used)
	}
}

func TestMissingInfoIsBlank(t *testing.T) {
	if i := From(context.Background()); i != (Info{}) {
		t.Fatalf("got %+v", i)
	}
}

// The book's wiring line, verbatim. As in the book,
// Move 89's ledger fields are left out here; the test
// above sets them.
func bookWiring(adapter llm.Model, name string,
	log *slog.Logger) llm.Model {
	m := Retrying{Next: Budgeted{Next: Traced{
		Next: adapter, Model: name, Log: log}}}
	return m
}

func TestBookWiringOrder(t *testing.T) {
	m := bookWiring(&adapter{}, "small", slog.Default())
	r, ok := m.(Retrying)
	if !ok {
		t.Fatal("retry must be outermost")
	}
	b, ok := r.Next.(Budgeted)
	if !ok {
		t.Fatal("budget must sit inside retry")
	}
	if _, ok := b.Next.(Traced); !ok {
		t.Fatal("tracer must sit next to the vendor")
	}
}
