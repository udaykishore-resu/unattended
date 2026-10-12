package shadow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

type memStore struct {
	mu    sync.Mutex
	pairs []Pair
	done  chan struct{}
}

func (m *memStore) Save(_ context.Context, p Pair) {
	m.mu.Lock()
	m.pairs = append(m.pairs, p)
	m.mu.Unlock()
	m.done <- struct{}{}
}

func TestLiveServesShadowLogs(t *testing.T) {
	live := &llm.FakeModel{Responses: []llm.Response{{Text: `{"lean":"dispute"}`}}}
	cand := &llm.FakeModel{Responses: []llm.Response{{Text: `{"lean":"pay"}`}}}
	st := &memStore{done: make(chan struct{}, 1)}
	s := &Shadow{Live: live, Cand: cand, Store: st, Sem: make(chan struct{}, 4)}

	// The request finishing must not kill the shadow call.
	ctx, cancel := context.WithCancel(context.Background())
	got, err := s.Do(ctx, "inv-40117", llm.Request{})
	cancel()
	if err != nil || got.Text != `{"lean":"dispute"}` {
		t.Fatalf("live = %q, %v", got.Text, err)
	}
	select {
	case <-st.done:
	case <-time.After(2 * time.Second):
		t.Fatal("shadow never saved")
	}
	p := st.pairs[0]
	// The scary bucket: clerk disputed, model said pay.
	if p.ID != "inv-40117" || p.Cand.Text != `{"lean":"pay"}` || p.Err != nil {
		t.Fatalf("pair = %+v", p)
	}
}

// slow blocks until released, to fill the semaphore.
type slow struct{ release chan struct{} }

func (s slow) Complete(context.Context, llm.Request) (llm.Response, error) {
	<-s.release
	return llm.Response{Text: "cand"}, nil
}

// When the shadow falls behind, the semaphore skips
// work; the live path never waits.
func TestShedsShadowLoad(t *testing.T) {
	live := &llm.FakeModel{}
	for range 5 {
		live.Responses = append(live.Responses, llm.Response{Text: "live"})
	}
	rel := make(chan struct{})
	st := &memStore{done: make(chan struct{}, 5)}
	s := &Shadow{Live: live, Cand: slow{rel}, Store: st, Sem: make(chan struct{}, 2)}
	start := time.Now()
	for i := range 5 {
		if _, err := s.Do(context.Background(), string(rune('a'+i)), llm.Request{}); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > time.Second {
		t.Fatal("live path waited on the shadow")
	}
	close(rel)
	for range 2 {
		<-st.done
	}
	time.Sleep(20 * time.Millisecond)
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.pairs) != 2 {
		t.Fatalf("shadowed %d, want 2 (3 shed)", len(st.pairs))
	}
}
