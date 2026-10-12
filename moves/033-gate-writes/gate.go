// Package gate runs read tools at once and turns
// every write into a proposal that waits for a yes.
//
// Move 33: reads run at once, writes wait for a yes.
package gate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

func (e *Executor) Do(ctx context.Context,
	c llm.ToolCall) (string, error) {
	ent, ok := e.Tools[c.Name]
	switch {
	case !ok:
		return "", errUnknownTool
	case e.Tainted && ent.Egress:
		return "", errTainted
	case ent.Kind == Read:
		return ent.Run(ctx, c.Input)
	}
	id, err := e.Queue.Propose(ctx, Proposal{
		RunID: e.RunID, CallID: c.ID,
		Tool: c.Name, Input: c.Input,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("queued for approval as %s."+
		" Do not call it again.", id), nil
}

// The types below are described in the book's prose:
// "The Executor holds the run ID and a Queue whose
// Propose stores the row. Its registry maps each tool
// name to an Entry: kind, egress flag and handler."

type Kind int

const (
	Read Kind = iota
	Write
)

type Entry struct {
	Kind   Kind
	Egress bool // input reaches another system
	Run    func(context.Context, json.RawMessage) (string, error)
}

type Proposal struct {
	RunID, CallID, Tool string
	Input               json.RawMessage
}

type Queue interface {
	Propose(context.Context, Proposal) (string, error)
}

type Executor struct {
	RunID   string
	Tainted bool // set once the run reads outside text
	Tools   map[string]Entry
	Queue   Queue
}

// ToolError is Move 32's error shape. Both error
// values below tell the model to stop.
type ToolError struct {
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	Fix   string `json:"fix"`
	Retry bool   `json:"retry"`
}

func (e *ToolError) Error() string { return e.Code + ": " + e.Msg }

var (
	errUnknownTool = &ToolError{Code: "unknown_tool",
		Msg: "no such tool for this agent",
		Fix: "do not retry; report it and stop"}
	errTainted = &ToolError{Code: "tainted",
		Msg: "this run read outside text; egress tools are off",
		Fix: "do not retry; finish without this tool"}
)
