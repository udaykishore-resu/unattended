package plan

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

// saves records every checkpoint. failAt makes the
// Nth Save fail (1-based), as a crashed or
// unreachable plan store would.
type saves struct {
	last   RunPlan
	n      int
	failAt int
	seen   []string // "step:status" per save
}

func (s *saves) Save(_ context.Context, p RunPlan) error {
	s.n++
	if s.n == s.failAt {
		return errors.New("plan store unavailable")
	}
	p.Steps = slices.Clone(p.Steps)
	s.last = p
	for _, st := range p.Steps {
		if st.Status != "" {
			s.seen = append(s.seen, st.Tool+":"+st.Status)
		}
	}
	return nil
}

type audit struct{ rows []string }

func (a *audit) Record(_ context.Context, approval string, c llm.ToolCall, _ error) {
	a.rows = append(a.rows, approval+" "+c.ID)
}

var tools6 = []string{"cancel_tender", "find_carrier",
	"send_tender", "set_pickup_window", "add_note", "notify_shipper"}

func rebook() RunPlan {
	var steps []Step
	for i, tool := range tools6 {
		steps = append(steps, Step{N: i + 1, Tool: tool,
			Input: json.RawMessage(`{"load_id":"HF-90412"}`)})
	}
	return approve(RunPlan{RunID: "run-9", Steps: steps})
}

// approve is what the approval path does when Dana
// says yes: ID, hash of the steps, deadline.
func approve(p RunPlan) RunPlan {
	p.Approval = "appr-dana-1"
	p.Hash = HashSteps(p.Steps)
	p.Expires = time.Now().Add(time.Hour)
	return p
}

func TestResumeAfterOutage(t *testing.T) {
	calls := map[string]int{}
	down := true
	tools := map[string]Entry{}
	for _, name := range tools6 {
		tools[name] = Entry{Run: func(context.Context, json.RawMessage) (string, error) {
			calls[name]++
			if name == "send_tender" && down {
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

	// The API recovers; resume from the saved plan. The
	// statuses changed, the approved hash still matches.
	down = false
	if err := Execute(context.Background(), ps, ex, ps.last); err != nil {
		t.Fatal(err)
	}
	if calls["cancel_tender"] != 1 || calls["find_carrier"] != 1 || calls["send_tender"] != 2 {
		t.Fatalf("finished steps reran: %v", calls)
	}
	// Every audit row carries Dana's approval and a stable call ID.
	if !strings.HasPrefix(au.rows[0], "appr-dana-1 run-9/step-1/") {
		t.Fatalf("audit = %v", au.rows)
	}
}

// A failing step is saved as "failed" and never,
// even for a moment, as "done".
func TestFailedStepNeverSavedAsDone(t *testing.T) {
	ex := &Executor{Audit: &audit{}, Tools: map[string]Entry{}}
	for _, name := range tools6 {
		ex.Tools[name] = Entry{Run: func(context.Context, json.RawMessage) (string, error) {
			if name == "find_carrier" {
				return "partial", errors.New("no carrier")
			}
			return "ok", nil
		}}
	}
	ps := &saves{}
	Execute(context.Background(), ps, ex, rebook())
	if slices.Contains(ps.seen, "find_carrier:done") ||
		!slices.Contains(ps.seen, "find_carrier:failed") ||
		ps.last.Steps[1].Result != "no carrier" {
		t.Fatalf("saves = %v, result = %q", ps.seen, ps.last.Steps[1].Result)
	}
}

// downstream is a carrier API that dedups on the
// idempotency key, as Move 52 asks. effects counts
// real side effects per tool.
type downstream struct {
	honorKey bool
	done     map[string]string
	effects  map[string]int
}

func (d *downstream) handler(name string) Entry {
	return Entry{Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
		k := Key(ctx)
		if k == "" {
			return "", errors.New("no idempotency key")
		}
		if out, ok := d.done[k]; ok && d.honorKey {
			return out, nil // replay, no new side effect
		}
		d.effects[name]++
		d.done[k] = "ok"
		return "ok", nil
	}}
}

// The crash window: step 2's write succeeds, then
// the checkpoint save fails. Execute stops. On
// resume step 2 runs again with the same key. Only
// a downstream that honours the key keeps it to one
// side effect; one that ignores it duplicates.
func TestCrashWindowNeedsDownstreamIdempotency(t *testing.T) {
	for _, honor := range []bool{true, false} {
		d := &downstream{honorKey: honor, done: map[string]string{}, effects: map[string]int{}}
		ex := &Executor{Audit: &audit{}, Tools: map[string]Entry{}}
		for _, name := range tools6 {
			ex.Tools[name] = d.handler(name)
		}
		ps := &saves{failAt: 2} // step 2's checkpoint fails
		err := Execute(context.Background(), ps, ex, rebook())
		if err == nil || ps.last.Steps[1].Status != "" {
			t.Fatalf("want stop with step 2 unsaved: %v %+v", err, ps.last.Steps[1])
		}
		if err := Execute(context.Background(), ps, ex, ps.last); err != nil {
			t.Fatal(err)
		}
		want := 1
		if !honor {
			want = 2
		}
		if d.effects["find_carrier"] != want || d.effects["cancel_tender"] != 1 {
			t.Fatalf("honor=%v effects=%v", honor, d.effects)
		}
	}
}

// Approval is bound to the exact steps and expires.
func TestApprovalBoundToPlan(t *testing.T) {
	ran := 0
	ex := &Executor{Audit: &audit{}, Tools: map[string]Entry{}}
	for _, name := range tools6 {
		ex.Tools[name] = Entry{Run: func(context.Context, json.RawMessage) (string, error) {
			ran++
			return "ok", nil
		}}
	}
	ctx := context.Background()

	edited := rebook()
	edited.Steps[4].Input = json.RawMessage(`{"load_id":"HF-11111"}`) // replan kept the old approval
	if err := Execute(ctx, &saves{}, ex, edited); !errors.Is(err, errChanged) {
		t.Fatalf("edited plan: %v", err)
	}
	added := rebook()
	added.Steps = append(added.Steps, Step{N: 7, Tool: "add_note"})
	if err := Execute(ctx, &saves{}, ex, added); !errors.Is(err, errChanged) {
		t.Fatalf("added step: %v", err)
	}
	stale := rebook()
	stale.Expires = time.Now().Add(-time.Minute)
	if err := Execute(ctx, &saves{}, ex, stale); !errors.Is(err, errStale) {
		t.Fatalf("stale plan: %v", err)
	}
	noDeadline := rebook()
	noDeadline.Expires = time.Time{} // missing deadline fails closed
	if err := Execute(ctx, &saves{}, ex, noDeadline); !errors.Is(err, errStale) {
		t.Fatalf("no deadline: %v", err)
	}
	if ran != 0 {
		t.Fatalf("%d steps ran on a bad approval", ran)
	}
}

// Same step, same key; an edited step, a new key.
func TestStepKeyFollowsInput(t *testing.T) {
	s := Step{N: 5, Tool: "add_note", Input: json.RawMessage(`{"text":"a"}`)}
	a, b := toCall("run-9", s), toCall("run-9", s)
	s.Input = json.RawMessage(`{"text":"b"}`)
	c := toCall("run-9", s)
	if a.ID != b.ID || a.ID == c.ID || !strings.HasPrefix(a.ID, "run-9/step-5/") {
		t.Fatalf("ids: %s %s %s", a.ID, b.ID, c.ID)
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
