// Package tracing wraps the model so every call
// logs prompt version, model, all four token counts,
// latency, stop reason and status.
//
// Move 83: trace every call.
package tracing

import (
	"context"
	"log/slog"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
	budget "github.com/udaykishore-resu/unattended/moves/089-token-budgets"
)

// Budgeted is Move 89's wrapper. It sits inside the
// retry layer, so every attempt is charged.
type Budgeted = budget.Budgeted

func (t Traced) Complete(ctx context.Context,
	r llm.Request) (llm.Response, error) {
	info := From(ctx)
	start := time.Now()
	resp, err := t.Next.Complete(ctx, r)
	status := "ok"
	if err != nil {
		status = "error"
	}
	u := resp.Usage
	t.Log.InfoContext(ctx, "llm.call",
		"trace_id", info.TraceID, "call_id", info.CallID,
		"attempt", info.Attempt,
		"workflow", info.Workflow,
		"prompt_version", info.Version, "model", t.Model,
		"in_tokens", u.InputTokens,
		"cache_read", u.CacheReadTokens,
		"cache_write", u.CacheWriteTokens,
		"out_tokens", u.OutputTokens,
		"latency_ms", time.Since(start).Milliseconds(),
		"stop", resp.StopReason, "status", status)
	return resp, err
}

// Traced sits next to the vendor adapter.
type Traced struct {
	Next  llm.Model
	Model string // what the router actually picked
	Log   *slog.Logger
}

// Info rides on the context: workflow, prompt
// version, trace ID and call ID. The retry layer
// stamps Attempt.
type Info struct {
	Workflow, Version, TraceID, CallID string
	Attempt                            int
}

type infoKey struct{}

// With puts Info on the context at the top of a run.
func With(ctx context.Context, i Info) context.Context {
	return context.WithValue(ctx, infoKey{}, i)
}

// From reads it back; a missing Info is all blanks,
// which the book says to count as a bug.
func From(ctx context.Context) Info {
	i, _ := ctx.Value(infoKey{}).(Info)
	return i
}

// Retrying runs Next.Complete inside the Retry
// function from Move 56. It is outermost, so it is
// the only retry layer, and it stamps the attempt
// number on the context.
type Retrying struct {
	Next       llm.Model
	Tries      int           // default 3
	Base, Ceil time.Duration // default 200ms, 5s
}

func (r Retrying) Complete(ctx context.Context,
	req llm.Request) (llm.Response, error) {
	tries, base, ceil := r.Tries, r.Base, r.Ceil
	if tries == 0 {
		tries = 3
	}
	if base == 0 {
		base, ceil = 200*time.Millisecond, 5*time.Second
	}
	var resp llm.Response
	attempt := 0
	err := retry.Retry(ctx, tries, base, ceil,
		func(ctx context.Context) error {
			attempt++
			i := From(ctx)
			i.Attempt = attempt
			var err error
			resp, err = r.Next.Complete(With(ctx, i), req)
			return err
		})
	return resp, err
}
