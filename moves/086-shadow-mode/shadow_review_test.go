package shadow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

// ctxStore records the context Save got, then
// behaves like a store: it waits on ctx when stuck.
type ctxStore struct {
	stuck bool
	got   chan saved
}

type saved struct {
	p        Pair
	deadline time.Time
	hasDL    bool
	errAtIn  error
}

func (c *ctxStore) Save(ctx context.Context, p Pair) {
	dl, ok := ctx.Deadline()
	s := saved{p: p, deadline: dl, hasDL: ok, errAtIn: ctx.Err()}
	if c.stuck {
		<-ctx.Done() // a stalled write that honors ctx
	}
	c.got <- s
}

// timesOut stands in for a candidate that hits its
// 30 s deadline: it checks the deadline is there and
// returns DeadlineExceeded at once.
type timesOut struct{ dl chan time.Duration }

func (t timesOut) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	d, ok := ctx.Deadline()
	if !ok {
		t.dl <- -1
	} else {
		t.dl <- time.Until(d)
	}
	return llm.Response{}, context.DeadlineExceeded
}

// A model timeout still gets its row, and the save
// runs under its own, fresh, bounded deadline.
func TestTimeoutStillSaved(t *testing.T) {
	live := &llm.FakeModel{Responses: []llm.Response{{Text: "live"}}}
	cand := timesOut{dl: make(chan time.Duration, 1)}
	st := &ctxStore{got: make(chan saved, 1)}
	s := &Shadow{Live: live, Cand: cand, Store: st, Sem: make(chan struct{}, 1)}
	if _, err := s.Do(context.Background(), "inv-1", llm.Request{}); err != nil {
		t.Fatal(err)
	}
	if d := <-cand.dl; d <= 25*time.Second || d > 30*time.Second {
		t.Fatalf("candidate deadline %v, want ~30s", d)
	}
	got := <-st.got
	if !errors.Is(got.p.Err, context.DeadlineExceeded) {
		t.Fatalf("pair err = %v", got.p.Err)
	}
	if got.errAtIn != nil {
		t.Fatalf("save ctx already done: %v", got.errAtIn)
	}
	if left := time.Until(got.deadline); !got.hasDL || left > 5*time.Second || left <= 0 {
		t.Fatalf("save deadline: has=%v left=%v; want <= 5s", got.hasDL, left)
	}
}

// A stuck store gives the slot back once its own
// deadline passes, so shadow work resumes.
func TestStuckStoreFreesSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the 5 s save deadline")
	}
	live := &llm.FakeModel{Responses: []llm.Response{{Text: "a"}, {Text: "b"}}}
	cand := &llm.FakeModel{Responses: []llm.Response{{Text: "c"}, {Text: "d"}}}
	st := &ctxStore{stuck: true, got: make(chan saved, 2)}
	s := &Shadow{Live: live, Cand: cand, Store: st, Sem: make(chan struct{}, 1)}
	if _, err := s.Do(context.Background(), "1", llm.Request{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-st.got:
	case <-time.After(8 * time.Second):
		t.Fatal("stuck save never returned: goroutine leaked")
	}
	// The slot is free again: the next request is shadowed.
	deadline := time.Now().Add(time.Second)
	for len(s.Sem) != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if len(s.Sem) != 0 {
		t.Fatal("slot still held")
	}
}

// The shadow only records the candidate's answer. Its
// tool calls land in the Pair; nothing executes them,
// and the caller only ever sees the live response.
func TestCandidateToolCallsOnlyLogged(t *testing.T) {
	live := &llm.FakeModel{Responses: []llm.Response{{Text: "live"}}}
	cand := &llm.FakeModel{Responses: []llm.Response{{ToolCalls: []llm.ToolCall{{
		ID: "t1", Name: "send_email", Input: json.RawMessage(`{"to":"x"}`)}}}}}
	st := &memStore{done: make(chan struct{}, 1)}
	s := &Shadow{Live: live, Cand: cand, Store: st, Sem: make(chan struct{}, 1)}
	got, err := s.Do(context.Background(), "inv-2", llm.Request{})
	if err != nil || got.Text != "live" || len(got.ToolCalls) != 0 {
		t.Fatalf("caller saw %+v, %v", got, err)
	}
	<-st.done
	if p := st.pairs[0]; len(p.Cand.ToolCalls) != 1 || p.Cand.ToolCalls[0].Name != "send_email" {
		t.Fatalf("pair = %+v", p)
	}
}

// A live failure is returned as is; no shadow call
// is made, so the comparison never has a blank side.
func TestLiveErrorSkipsShadow(t *testing.T) {
	live := &llm.FakeModel{Err: errors.New("429")}
	cand := &llm.FakeModel{}
	s := &Shadow{Live: live, Cand: cand, Store: &memStore{}, Sem: make(chan struct{}, 1)}
	if _, err := s.Do(context.Background(), "x", llm.Request{}); err == nil {
		t.Fatal("want live error")
	}
	time.Sleep(20 * time.Millisecond)
	if len(cand.Calls) != 0 {
		t.Fatal("shadow ran after a live failure")
	}
}
