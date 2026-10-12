package runcaps

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
)

func toolResp(u llm.Usage) llm.Response {
	return llm.Response{Usage: u, ToolCalls: []llm.ToolCall{{ID: "c",
		Name: "add_note", Input: json.RawMessage(`{}`)}}}
}

// The response that crosses the cap is charged
// before its tool calls run, so they never run.
func TestOverBudgetStartsNoTool(t *testing.T) {
	f := &llm.FakeModel{}
	for range 5 {
		f.Responses = append(f.Responses, toolResp(llm.Usage{InputTokens: 400}))
	}
	r := &Run{Limits: Limits{Steps: 100, Tokens: 1000}}
	ran := 0
	_, err := Loop(context.Background(), f, llm.Request{},
		func(context.Context, llm.ToolCall) (string, error) { ran++; return "ok", nil }, r, true)
	// 400, 800 run their tools; 1200 crosses the cap first.
	if !errors.Is(err, ErrBudget) || ran != 2 {
		t.Fatalf("err=%v ran=%d", err, ran)
	}
}

// A tool that honours ctx stops at the wall clock,
// and the loop reports why.
func TestWallClockStopsCooperativeTool(t *testing.T) {
	ctx, cancel := WithWall(context.Background(), Limits{Wall: 20 * time.Millisecond})
	defer cancel()
	f := &llm.FakeModel{Responses: []llm.Response{toolResp(llm.Usage{})}}
	r := &Run{Limits: Limits{Steps: 10, Tokens: 1 << 30}}
	start := time.Now()
	_, err := Loop(ctx, f, llm.Request{}, func(ctx context.Context, _ llm.ToolCall) (string, error) {
		<-ctx.Done() // a well-behaved slow tool
		return "", ctx.Err()
	}, r, true)
	if !errors.Is(err, ErrBudget) || time.Since(start) > time.Second {
		t.Fatalf("err=%v after %v", err, time.Since(start))
	}
}

// The wall clock is cooperative: a handler that
// ignores ctx runs to the end regardless. This test
// documents the limit; it is why the book says to
// lint tool code for context-free calls.
func TestWallClockCannotStopToolThatIgnoresCtx(t *testing.T) {
	ctx, cancel := WithWall(context.Background(), Limits{Wall: 10 * time.Millisecond})
	defer cancel()
	f := &llm.FakeModel{Responses: []llm.Response{toolResp(llm.Usage{})}}
	r := &Run{Limits: Limits{Steps: 10, Tokens: 1 << 30}}
	start := time.Now()
	Loop(ctx, f, llm.Request{}, func(context.Context, llm.ToolCall) (string, error) {
		time.Sleep(80 * time.Millisecond) // ignores ctx
		return "ok", nil
	}, r, true)
	if time.Since(start) < 80*time.Millisecond {
		t.Fatal("a ctx-ignoring tool was stopped early; the claim changed")
	}
}

// flaky fails its first n calls, then answers.
type flaky struct {
	mu    sync.Mutex
	fails int
	calls int
	u     llm.Usage
}

func (f *flaky) Complete(context.Context, llm.Request) (llm.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.fails {
		// A failed attempt can still report usage.
		return llm.Response{Usage: llm.Usage{InputTokens: 50}}, errors.New("503")
	}
	return llm.Response{Text: "ok", Usage: f.u, StopReason: llm.StopEnd}, nil
}

// Retry outside, charge inside: every attempt is a
// step and pays the usage it reported, all four
// fields included.
func TestEveryAttemptPays(t *testing.T) {
	r := &Run{Limits: Limits{Steps: 10, Tokens: 1 << 30}}
	f := &flaky{fails: 2, u: llm.Usage{InputTokens: 10, CacheReadTokens: 300,
		CacheWriteTokens: 20, OutputTokens: 40}}
	m := ChargeEach{Next: f, Run: r}
	err := retry.Retry(context.Background(), 5, time.Millisecond, 2*time.Millisecond,
		func(ctx context.Context) error { _, err := m.Complete(ctx, llm.Request{}); return err })
	if err != nil {
		t.Fatal(err)
	}
	if r.steps != 3 || r.tokens != 2*50+(10+300+20+40) {
		t.Fatalf("steps=%d tokens=%d", r.steps, r.tokens)
	}
}

// Once the budget is spent inside the retry layer,
// the retries stop: no more model calls.
func TestSpentBudgetStopsRetries(t *testing.T) {
	r := &Run{Limits: Limits{Steps: 2, Tokens: 1 << 30}}
	f := &flaky{fails: 100}
	m := ChargeEach{Next: f, Run: r}
	err := retry.Retry(context.Background(), 5, time.Millisecond, 2*time.Millisecond,
		func(ctx context.Context) error { _, err := m.Complete(ctx, llm.Request{}); return err })
	if !errors.Is(err, ErrBudget) || f.calls != 2 {
		t.Fatalf("err=%v calls=%d", err, f.calls)
	}
}
