// Package loopguard catches the exact same tool call
// twice in one run.
//
// Move 41: loop detection: same tool, same args,
// twice. The first repeat (the second identical
// call) gets a ToolError; the second repeat (the
// third identical call) ends the run with ErrLoop.
package loopguard

import (
	"encoding/json"
	"errors"

	"github.com/udaykishore-resu/unattended/llm"
)

var ErrLoop = errors.New("loop: same call 3 times")

type LoopGuard struct{ seen map[string]int }

func (g *LoopGuard) Check(c llm.ToolCall) error {
	var v any
	if json.Unmarshal(c.Input, &v) != nil {
		v = string(c.Input)
	}
	b, _ := json.Marshal(v) // map keys sorted
	k := c.Name + "\x00" + string(b)
	if g.seen == nil {
		g.seen = map[string]int{}
	}
	g.seen[k]++
	switch g.seen[k] {
	case 1:
		return nil
	case 2:
		return &ToolError{Code: "repeat_call",
			Msg: "you already made this exact call",
			Fix: "use the earlier result or stop"}
	}
	return ErrLoop
}

// ToolError is Move 32's error shape.
type ToolError struct {
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	Fix   string `json:"fix"`
	Retry bool   `json:"retry"`
}

func (e *ToolError) Error() string { return e.Code + ": " + e.Msg }
