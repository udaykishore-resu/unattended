package runcaps

import (
	"context"
	"errors"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
)

// RunFunc runs one tool call (Move 32's executor).
type RunFunc func(context.Context,
	llm.ToolCall) (string, error)

// Loop is Move 32's loop with Charge in place of the
// fixed step count: Charge runs after each model
// response and before any tool, so a run over budget
// never starts another tool call. The cap error goes
// to the caller, not the model. ctx should come from
// WithWall; read the reason back with context.Cause.
//
// If m is wrapped in ChargeEach (retries inside),
// pass charge=false so attempts are not counted
// twice.
func Loop(ctx context.Context, m llm.Model,
	req llm.Request, run RunFunc, r *Run,
	charge bool) (string, error) {
	for {
		resp, err := m.Complete(ctx, req)
		if err != nil {
			if ctx.Err() != nil {
				return "", context.Cause(ctx)
			}
			return "", err
		}
		if charge {
			if err := r.Charge(resp.Usage); err != nil {
				return "", err
			}
		}
		switch resp.StopReason {
		case llm.StopEnd:
			return resp.Text, nil
		case llm.StopToolUse:
		default:
			return "", fmt.Errorf("stopped: %s",
				resp.StopReason)
		}
		req.Messages = append(req.Messages, llm.Message{
			Role: "assistant", Content: resp.Text,
			ToolCalls: resp.ToolCalls})
		for _, c := range resp.ToolCalls {
			if ctx.Err() != nil {
				return "", context.Cause(ctx)
			}
			out, err := run(ctx, c)
			if err != nil { // Move 32's toolResult, trimmed
				out = `{"code":"internal","retry":false}`
			}
			req.Messages = append(req.Messages,
				llm.Message{Role: "tool", ToolCallID: c.ID,
					Content: out})
		}
	}
}

// ChargeEach charges every model attempt, failed
// ones included, so put it INSIDE the retry layer:
// retry(ChargeEach(model)). A failed attempt costs a
// step and whatever usage it reported. Once the
// budget is spent it returns a Permanent error, so
// the retry layer stops instead of calling again.
type ChargeEach struct {
	Next llm.Model
	Run  *Run
}

func (c ChargeEach) Complete(ctx context.Context,
	req llm.Request) (llm.Response, error) {
	resp, err := c.Next.Complete(ctx, req)
	if cerr := c.Run.Charge(resp.Usage); cerr != nil {
		return llm.Response{}, retry.Permanent{
			Err: errors.Join(cerr, err)}
	}
	return resp, err
}
