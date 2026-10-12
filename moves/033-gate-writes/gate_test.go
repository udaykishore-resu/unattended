package gate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

type memQueue struct{ got []Proposal }

func (q *memQueue) Propose(_ context.Context, p Proposal) (string, error) {
	q.got = append(q.got, p)
	return "prop-1", nil
}

func newExec(writes *int) (*Executor, *memQueue) {
	q := &memQueue{}
	run := func(context.Context, json.RawMessage) (string, error) { return "ok", nil }
	return &Executor{RunID: "run-7", Queue: q, Tools: map[string]Entry{
		"get_load":  {Kind: Read, Run: run},
		"fetch_url": {Kind: Read, Egress: true, Run: run},
		"set_status": {Kind: Write, Run: func(context.Context, json.RawMessage) (string, error) {
			*writes++
			return "done", nil
		}},
	}}, q
}

func TestReadsRunWritesQueue(t *testing.T) {
	writes := 0
	e, q := newExec(&writes)
	ctx := context.Background()
	if out, err := e.Do(ctx, llm.ToolCall{ID: "c1", Name: "get_load"}); out != "ok" || err != nil {
		t.Fatalf("read: %q %v", out, err)
	}
	out, err := e.Do(ctx, llm.ToolCall{ID: "c2", Name: "set_status",
		Input: json.RawMessage(`{"status":"cancelled"}`)})
	if err != nil || !strings.Contains(out, "Do not call it again") {
		t.Fatalf("write: %q %v", out, err)
	}
	// Nothing changed: the handler never ran, a proposal row exists.
	if writes != 0 || len(q.got) != 1 || q.got[0].RunID != "run-7" ||
		string(q.got[0].Input) != `{"status":"cancelled"}` {
		t.Fatalf("writes=%d proposals=%+v", writes, q.got)
	}
}

func TestTaintedRunLosesEgress(t *testing.T) {
	writes := 0
	e, _ := newExec(&writes)
	e.Tainted = true
	_, err := e.Do(context.Background(), llm.ToolCall{Name: "fetch_url"})
	var te *ToolError
	if !errors.As(err, &te) || te.Code != "tainted" {
		t.Fatalf("got %v", err)
	}
	// Closed reads still work.
	if out, _ := e.Do(context.Background(), llm.ToolCall{Name: "get_load"}); out != "ok" {
		t.Fatal("closed read blocked")
	}
}

func TestUnknownTool(t *testing.T) {
	writes := 0
	e, _ := newExec(&writes)
	_, err := e.Do(context.Background(), llm.ToolCall{Name: "cancel_load"})
	if !errors.Is(err, errUnknownTool) {
		t.Fatalf("got %v", err)
	}
}

type downQueue struct{}

func (downQueue) Propose(context.Context, Proposal) (string, error) {
	return "", errors.New("queue unavailable")
}

// The queue is down: Propose fails, the write does
// not run, and the error goes back (as Move 32's
// fixed "do not retry; report it and stop").
func TestQueueDownWriteDoesNotRun(t *testing.T) {
	writes := 0
	e, _ := newExec(&writes)
	e.Queue = downQueue{}
	out, err := e.Do(context.Background(), llm.ToolCall{ID: "c1", Name: "set_status"})
	if err == nil || out != "" || writes != 0 {
		t.Fatalf("out=%q err=%v writes=%d", out, err, writes)
	}
}
