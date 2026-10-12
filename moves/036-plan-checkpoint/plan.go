// Package plan executes an approved plan step by
// step, saving progress after each.
//
// Move 36: plan, checkpoint, then execute.
package plan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

func Execute(ctx context.Context, ps PlanStore,
	ex *Executor, p RunPlan) error {
	if err := p.CheckApproval(time.Now()); err != nil {
		return err // unapproved, stale or edited
	}
	for i := range p.Steps {
		s := &p.Steps[i]
		if s.Status == "done" {
			continue
		}
		out, err := ex.DoApproved(ctx, p.Approval,
			toCall(p.RunID, *s))
		if err != nil {
			s.Status, s.Result = "failed", err.Error()
		} else {
			s.Status, s.Result = "done", out
		}
		if serr := ps.Save(ctx, p); serr != nil {
			return serr // resume reruns s, same key
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
	ctx = WithKey(ctx, c.ID) // handlers pass it on
	out, err := ent.Run(ctx, c.Input)
	e.Audit.Record(ctx, approval, c, err)
	return out, err
}

// The rest is described in the prose; minimal.

// RunPlan is the model's plan, approved as a whole.
// When Dana approves, the approval path stores the
// approval ID, Hash = HashSteps(p.Steps) and an
// expiry.
type RunPlan struct {
	RunID    string
	Approval string // approval ID; empty = not approved
	Hash     string // HashSteps at approval time
	Expires  time.Time
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

// CheckApproval refuses a plan with no approval, an
// approval past its deadline, or steps that no
// longer hash to what was approved. Execute calls it
// on every start, including every resume.
func (p RunPlan) CheckApproval(now time.Time) error {
	switch {
	case p.Approval == "":
		return errNotApproved
	case !now.Before(p.Expires):
		return errStale
	case HashSteps(p.Steps) != p.Hash:
		return errChanged
	}
	return nil
}

// HashSteps covers what was approved: each step's
// number, tool and input. Status and Result are left
// out, because they change as the plan runs.
func HashSteps(steps []Step) string {
	h := sha256.New()
	for _, s := range steps {
		fmt.Fprintf(h, "%d\x00%s\x00%d\x00", s.N, s.Tool, len(s.Input))
		h.Write(s.Input)
	}
	return hex.EncodeToString(h.Sum(nil))
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
	errStale       = errors.New("approval expired; replan")
	errChanged     = errors.New("plan changed since approval")
	errUnknownTool = errors.New("unknown tool")
)

// toCall builds the call ID from the run ID, the
// step number and a short hash of the step's tool
// and input: a stable idempotency key (Move 52). A
// rerun of the same step reuses it; an edited step
// gets a new one, so a downstream system can't
// replay an old result for new input.
func toCall(runID string, s Step) llm.ToolCall {
	sum := sha256.Sum256(append([]byte(s.Tool+"\x00"),
		s.Input...))
	return llm.ToolCall{
		ID: fmt.Sprintf("%s/step-%d/%x", runID, s.N,
			sum[:6]),
		Name:  s.Tool,
		Input: s.Input,
	}
}

type keyCtx struct{}

// WithKey puts the step's idempotency key in ctx.
// Write handlers must pass Key(ctx) to the system
// they call (as an Idempotency-Key header, a unique
// column, Move 52's Once). A handler that drops it
// gets a duplicate side effect in the crash window.
func WithKey(ctx context.Context, k string) context.Context {
	return context.WithValue(ctx, keyCtx{}, k)
}

// Key returns the idempotency key DoApproved set.
func Key(ctx context.Context) string {
	k, _ := ctx.Value(keyCtx{}).(string)
	return k
}
