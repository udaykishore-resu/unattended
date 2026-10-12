package plan

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

type saves struct{ last RunPlan }

func (s *saves) Save(_ context.Context, p RunPlan) error {
	p.Steps = slices.Clone(p.Steps)
	s.last = p
	return nil
}

type audit struct{ rows []string }

func (a *audit) Record(_ context.Context, approval string, c llm.ToolCall, _ error) {
	a.rows = append(a.rows, approval+" "+c.ID)
}

func rebook() RunPlan {
	var steps []Step
	for i, tool := range []string{"cancel_tender", "find_carrier",
		"send_tender", "set_pickup_window", "add_note", "notify_shipper"} {
		steps = append(steps, Step{N: i + 1, Tool: tool, Input: json.RawMessage(`{}`)})
	}
	return RunPlan{RunID: "run-9", Approval: "appr-dana-1", Steps: steps}
}

func TestResumeAfterOutage(t *testing.T) {
	calls := map[string]int{}
	down := true
	tools := map[string]Entry{}
	for _, s := range rebook().Steps {
		tools[s.Tool] = Entry{Run: func(context.Context, json.RawMessage) (string, error) {
			calls[s.Tool]++
			if s.Tool == "send_tender" && down {
				return "", errors.New("carrier API down")
			}
			return "ok", nil
		}}
	}
	ps, au := &saves{}, &audit{}
	ex := &Executor{Tools: tools, Audit: au}

	// First run: step 3 fails; steps 1-2 are checkpointed as done.
	if err := Execute(context.Background(), ps, ex, rebook()); err == nil {
		t.Fatal("want the outage error")
	}
	if ps.last.Steps[1].Status != "done" || ps.last.Steps[2].Status != "failed" {
		t.Fatalf("checkpoint = %+v", ps.last.Steps[:3])
	}

	// The API recovers; resume from the saved plan.
	down = false
	if err := Execute(context.Background(), ps, ex, ps.last); err != nil {
		t.Fatal(err)
	}
	if calls["cancel_tender"] != 1 || calls["find_carrier"] != 1 || calls["send_tender"] != 2 {
		t.Fatalf("finished steps reran: %v", calls)
	}
	// Every audit row carries Dana's approval and a stable call ID.
	if au.rows[0] != "appr-dana-1 run-9/step-1" {
		t.Fatalf("audit = %v", au.rows)
	}
}

func TestNotApproved(t *testing.T) {
	p := rebook()
	p.Approval = ""
	if err := Execute(context.Background(), &saves{}, &Executor{}, p); !errors.Is(err, errNotApproved) {
		t.Fatalf("got %v", err)
	}
}

func TestAllowListStillHolds(t *testing.T) {
	ex := &Executor{Tools: map[string]Entry{}, Audit: &audit{}}
	_, err := ex.DoApproved(context.Background(), "appr", llm.ToolCall{Name: "cancel_load"})
	if !errors.Is(err, errUnknownTool) {
		t.Fatalf("got %v", err)
	}
}
