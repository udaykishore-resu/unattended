// Package toolerrors returns tool failures the model
// can act on, and runs the full tool loop.
//
// Move 32: return errors the model can act on.
package toolerrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

// ToolError is a failure the model can act on.
type ToolError struct {
	Code  string `json:"code"`  // not_found, invalid...
	Msg   string `json:"msg"`   // what happened
	Fix   string `json:"fix"`   // what to try next
	Retry bool   `json:"retry"` // same call may work
}

func (e *ToolError) Error() string {
	return e.Code + ": " + e.Msg
}

func toolResult(out string, err error) string {
	var te *ToolError
	if err == nil {
		return out
	}
	if errors.As(err, &te) {
		b, _ := json.Marshal(te)
		return string(b)
	}
	return `{"code":"internal","retry":false,` +
		`"fix":"do not retry; report it and stop"}`
}

type RunFunc func(context.Context,
	llm.ToolCall) (string, error)

func Loop(ctx context.Context, m llm.Model,
	req llm.Request, run RunFunc) (string, error) {
	for step := 0; step < 20; step++ {
		resp, err := m.Complete(ctx, req)
		if err != nil {
			return "", err
		}
		switch resp.StopReason {
		case llm.StopEnd:
			return resp.Text, nil
		case llm.StopToolUse:
		default: // max_tokens, refusal, other
			return "", fmt.Errorf("stopped: %s",
				resp.StopReason)
		}
		req.Messages = append(req.Messages, llm.Message{
			Role: "assistant", Content: resp.Text,
			ToolCalls: resp.ToolCalls})
		for _, c := range resp.ToolCalls {
			out, err := run(ctx, c)
			req.Messages = append(req.Messages,
				llm.Message{Role: "tool", ToolCallID: c.ID,
					Content: toolResult(out, err)})
		}
	}
	return "", errors.New("step cap reached")
}
