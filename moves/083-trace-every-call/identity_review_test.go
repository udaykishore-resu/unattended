package tracing

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	shadow "github.com/udaykishore-resu/unattended/moves/086-shadow-mode"
	budget "github.com/udaykishore-resu/unattended/moves/089-token-budgets"
)

// syncBuf lets the shadow goroutine log safely.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

type savedOnce chan struct{}

func (s savedOnce) Save(context.Context, shadow.Pair) { s <- struct{}{} }

func stack(next llm.Model, name string, log *slog.Logger,
	l *budget.Ledger) llm.Model {
	return Retrying{Base: time.Millisecond, Ceil: time.Millisecond,
		Next: Budgeted{Ledger: l, EstIn: budget.EstBytes,
			Key:  func(ctx context.Context) string { return From(ctx).Workflow },
			Next: Traced{Next: next, Model: name, Log: log}}}
}

// Every call in one run carries the same identity:
// the first attempt, its retry, the repair call, the
// escalation to the large model and the shadow call.
func TestOneRunOneIdentity(t *testing.T) {
	var buf syncBuf
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	l := budget.NewLedger(map[string]int{"ops-triage": 1_000_000})
	small := stack(&adapter{}, "small", log, l) // times out once
	large := stack(&llm.FakeModel{Responses: []llm.Response{
		{Text: "esc"}, {Text: "shadow"}}}, "large", log, l)

	ctx := With(context.Background(), Info{Workflow: "ops-triage",
		Version: "3f9a1c2b7d4e", TraceID: "run-77", CallID: "c-1"})
	req := llm.Request{System: "sys", MaxTokens: 10}
	for _, m := range []llm.Model{small, small, large} { // first, repair, escalation
		if _, err := m.Complete(ctx, req); err != nil && !errors.Is(err, llm.ErrNoResponses) {
			t.Fatal(err)
		}
	}
	done := make(savedOnce, 1)
	sh := &shadow.Shadow{Live: &llm.FakeModel{Responses: []llm.Response{{Text: "live"}}},
		Cand: large, Store: done, Sem: make(chan struct{}, 1)}
	reqCtx, cancel := context.WithCancel(ctx)
	if _, err := sh.Do(reqCtx, "inv-1", req); err != nil {
		t.Fatal(err)
	}
	cancel()
	<-done

	buf.mu.Lock()
	got := lines(t, &buf.b)
	buf.mu.Unlock()
	if len(got) != 5 { // 2 attempts + repair + escalation + shadow
		t.Fatalf("got %d lines: %v", len(got), got)
	}
	models := map[string]int{}
	for i, g := range got {
		if g["trace_id"] != "run-77" || g["workflow"] != "ops-triage" ||
			g["prompt_version"] != "3f9a1c2b7d4e" {
			t.Errorf("line %d lost the run identity: %v", i, g)
		}
		models[g["model"].(string)]++
	}
	if models["small"] != 3 || models["large"] != 2 {
		t.Fatalf("models = %v", models)
	}
}

// A spent budget inside the stack is tried once: the
// refusal is Permanent, and the adapter is never called.
func TestBudgetRefusalInStackTriedOnce(t *testing.T) {
	var buf bytes.Buffer
	ad := &adapter{}
	m := stack(ad, "small", slog.New(slog.NewJSONHandler(&buf, nil)),
		budget.NewLedger(map[string]int{"ops-triage": 1}))
	ctx := With(context.Background(), Info{Workflow: "ops-triage"})
	_, err := m.Complete(ctx, llm.Request{MaxTokens: 100})
	if !errors.Is(err, budget.ErrBudget) || ad.calls != 0 {
		t.Fatalf("err=%v adapter calls=%d", err, ad.calls)
	}
}

// Swap the order and the synthesis's warning comes
// true. Budget outside retry: the failed attempt is
// never charged. Tracer outside retry: one line for
// two attempts.
func TestSwappedOrderLosesData(t *testing.T) {
	ctx := With(context.Background(), Info{Workflow: "ops-triage"})
	req := llm.Request{MaxTokens: 100}

	l := budget.NewLedger(map[string]int{"ops-triage": 1_000_000})
	var b1 bytes.Buffer
	budgetOutside := Budgeted{Ledger: l, EstIn: budget.EstBytes,
		Key: func(ctx context.Context) string { return From(ctx).Workflow },
		Next: Retrying{Base: time.Millisecond, Ceil: time.Millisecond,
			Next: Traced{Next: &adapter{}, Model: "small",
				Log: slog.New(slog.NewJSONHandler(&b1, nil))}}}
	if _, err := budgetOutside.Complete(ctx, req); err != nil {
		t.Fatal(err)
	}
	if used := l.Used("ops-triage"); used != 120+6000+40 {
		t.Fatalf("used = %d; expected only the success charged", used)
	}

	var b2 bytes.Buffer
	tracerOutside := Traced{Model: "small", Log: slog.New(slog.NewJSONHandler(&b2, nil)),
		Next: Retrying{Base: time.Millisecond, Ceil: time.Millisecond, Next: &adapter{}}}
	if _, err := tracerOutside.Complete(ctx, req); err != nil {
		t.Fatal(err)
	}
	if n := len(lines(t, &b2)); n != 1 {
		t.Fatalf("lines = %d; expected the retry to vanish", n)
	}
}
