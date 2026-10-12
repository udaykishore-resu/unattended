// Package loopguard catches the exact same tool call
// twice in one run.
//
// Move 41: loop detection: same tool, same args,
// twice. The first repeat (the second identical
// call) gets a ToolError; the second repeat (the
// third identical call) ends the run with ErrLoop.
//
// key is the tool name plus its arguments,
// re-encoded so key order and spacing don't count.
package loopguard

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/udaykishore-resu/unattended/llm"
)

var ErrLoop = errors.New("loop: third identical call")

func key(c llm.ToolCall) (string, error) {
	d := json.NewDecoder(bytes.NewReader(c.Input))
	d.UseNumber() // big numeric IDs stay exact
	var v any
	if d.Decode(&v) != nil {
		v = string(c.Input) // not JSON: raw text
	}
	b, err := json.Marshal(v) // map keys sorted
	return c.Name + "\x00" + string(b), err
}

type LoopGuard struct {
	mu   sync.Mutex
	seen map[string]int
}

func (g *LoopGuard) Check(c llm.ToolCall) error {
	k, err := key(c)
	if err != nil {
		return err // can't key it: end the run
	}
	g.mu.Lock()
	defer g.mu.Unlock()
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
