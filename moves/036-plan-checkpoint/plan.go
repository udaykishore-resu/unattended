// Package plan executes an approved plan step by
// step, saving progress after each.
//
// Move 36: plan, checkpoint, then execute.
package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

func Execute(ctx context.Context, ps PlanStore,
	ex *Executor, p RunPlan) error {
	if p.Approval == "" {
		return errNotApproved
	}
	for i := range p.Steps {
		s := &p.Steps[i]
		if s.Status == "done" {
			continue
		}
		out, err := ex.DoApproved(ctx, p.Approval,
			toCall(p.RunID, *s))
		s.Status, s.Result = "done", out
		if err != nil {
			s.Status, s.Result = "failed", err.Error()
		}
		if serr := ps.Save(ctx, p); serr != nil {
			return serr
		}
		if err != nil {
			return err // stop: a human or a replan
		}
	}
	return nil
}

func (e *Executor) DoApproved(ctx context.Context,
	approval string, c llm.ToolCall) (string, error) {
	ent, ok := e.Tools[c.Name] // allow-list still holds
	if !ok {
		return "", errUnknownTool
	}
	out, err := ent.Run(ctx, c.Input)
	e.Audit.Record(ctx, approval, c, err)
	return out, err
}

// The rest is implied by the prose; minimal.

// RunPlan is the model's plan, approved as a whole.
type RunPlan struct {
	RunID    string
	Approval string // approval ID; empty = not approved
	Steps    []Step
}

// Step is one tool call with its reason and status.
type Step struct {
	N      int // step number, part of the call ID
	Tool   string
	Input  json.RawMessage
	Reason string
	Status string // "", "done", "failed"
	Result string
}

type PlanStore interface {
	Save(ctx context.Context, p RunPlan) error
}

// Audit writes the usual audit row (Move 97),
// stamped with the approval ID.
type Audit interface {
	Record(ctx context.Context, approval string,
		c llm.ToolCall, err error)
}

// Entry is the registry row from Move 33.
type Entry struct {
	Run func(context.Context, json.RawMessage) (string, error)
}

// Executor is Move 33's executor, trimmed to what
// DoApproved needs.
type Executor struct {
	Tools map[string]Entry
	Audit Audit
}

var (
	errNotApproved = errors.New("plan not approved")
	errUnknownTool = errors.New("unknown tool")
)

// toCall builds the call ID from the run ID and step
// number: a stable idempotency key (Move 52).
func toCall(runID string, s Step) llm.ToolCall {
	return llm.ToolCall{
		ID:    fmt.Sprintf("%s/step-%d", runID, s.N),
		Name:  s.Tool,
		Input: s.Input,
	}
}
