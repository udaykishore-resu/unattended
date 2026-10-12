// Package structured reads the JSON a model produced
// through native structured output or a forced tool
// call, after checking the stop reason.
//
// Move 14: force the format with native structured
// output.
package structured

import (
	"errors"

	"github.com/udaykishore-resu/unattended/llm"
)

var (
	ErrCutOff  = errors.New("llm: reply cut off")
	ErrRefused = errors.New("llm: model refused")
	ErrStopped = errors.New("llm: unexpected stop")
)

// output returns the JSON the model produced, from
// native structured output or a forced tool call.
func output(r llm.Response) ([]byte, error) {
	switch r.StopReason {
	case llm.StopEnd, llm.StopToolUse:
	case llm.StopMaxTokens:
		return nil, ErrCutOff
	case llm.StopRefusal:
		return nil, ErrRefused
	default:
		return nil, ErrStopped
	}
	if len(r.ToolCalls) == 1 {
		return r.ToolCalls[0].Input, nil
	}
	return []byte(r.Text), nil
}
