package durable

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
	idem "github.com/udaykishore-resu/unattended/moves/052-idempotency-keys"
)

// journal emulates wf_steps (primary key on run and
// step) and wf_runs.
type journal struct {
	mu      sync.Mutex
	rows    map[string][]byte
	runs    map[string]runRow
	failPut int // fail the next n putStep calls
}

type runRow struct {
	owner string
	until time.Time
}

func newJournal() *journal {
	return &journal{rows: map[string][]byte{}, runs: map[string]runRow{}}
}

func (j *journal) h(q string, args []any) ([]string, [][]any, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if q == claimRunQ {
		id, owner, until := args[0].(string), args[1].(string), args[2].(time.Time)
		cur, ok := j.runs[id]
		if ok && cur.until.After(time.Now()) && cur.owner != owner {
			return nil, nil, nil
		}
		j.runs[id] = runRow{owner, until}
		return nil, [][]any{{1}}, nil
	}
	k := args[0].(string) + "/" + args[1].(string)
	switch q {
	case getStep:
		if b, ok := j.rows[k]; ok {
			return []string{"result"}, [][]any{{b}}, nil
		}
		return []string{"result"}, nil, nil // sql.ErrNoRows
	case putStep:
		if j.failPut > 0 {
			j.failPut--
			return nil, nil, errors.New("db: connection reset")
		}
		if _, dup := j.rows[k]; dup {
			return nil, nil, errors.New(`duplicate key value violates unique constraint "wf_steps_pkey"`)
		}
		j.rows[k] = args[2].([]byte)
		return nil, [][]any{{1}}, nil
	}
	return nil, nil, errors.New("unknown query")
}

type quote struct {
	Carrier string `json:"carrier"`
	Cents   int64  `json:"cents"`
}

// mailer is a carrier email sender. If it honors the
// key, a second send with the same key is a no-op.
type mailer struct {
	mu     sync.Mutex
	honors bool
	sent   map[string]int
	total  int
}

func (m *mailer) send(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.honors && m.sent[key] > 0 {
		return
	}
	m.sent[key]++
	m.total++
}

// The quote flow: ask three carriers, then book.
func quoteFlow(ctx context.Context, r *Run, m *mailer, crashAfter int) error {
	for i, c := range []string{"RDX", "Blue Mesa", "Okafor"} {
		if i == crashAfter {
			return errors.New("node drained")
		}
		_, err := Step(ctx, r, fmt.Sprintf("ask-carrier-%d", i),
			func(_ context.Context, key string) (quote, error) {
				m.send(key) // the side effect
				return quote{c, int64(200000 + i*1000)}, nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// A node drain at hour four must not resend the
// carrier emails.
func TestReplayReturnsJournaledResults(t *testing.T) {
	j := newJournal()
	r := &Run{ID: "quote-77", DB: sqlfake.Open(j.h)}
	m := &mailer{sent: map[string]int{}}
	ctx := context.Background()

	if err := quoteFlow(ctx, r, m, 2); err == nil {
		t.Fatal("want the drain")
	}
	if err := quoteFlow(ctx, r, m, -1); err != nil {
		t.Fatal(err)
	}
	if m.total != 3 {
		t.Fatalf("emails sent = %d, want 3 (one per carrier)", m.total)
	}
	// The journaled value comes back typed.
	q, err := Step(ctx, r, "ask-carrier-1", func(context.Context, string) (quote, error) {
		t.Fatal("must replay from the journal")
		return quote{}, nil
	})
	if err != nil || q.Carrier != "Blue Mesa" || q.Cents != 201000 {
		t.Fatalf("q=%+v err=%v", q, err)
	}
}

func TestStepErrorIsNotJournaled(t *testing.T) {
	j := newJournal()
	r := &Run{ID: "r", DB: sqlfake.Open(j.h)}
	_, err := Step(context.Background(), r, "s", func(context.Context, string) (int, error) {
		return 0, errors.New("carrier API 503")
	})
	if err == nil || len(j.rows) != 0 {
		t.Fatalf("err=%v rows=%d", err, len(j.rows))
	}
}

// The step gets Move 52's key from step name and run
// ID: the same on every replay, different per step
// and per run.
func TestStepKey(t *testing.T) {
	j := newJournal()
	db := sqlfake.Open(j.h)
	var got []string
	for _, id := range []string{"run-1", "run-2"} {
		Step(context.Background(), &Run{ID: id, DB: db}, "ask-carrier-0",
			func(_ context.Context, k string) (int, error) {
				got = append(got, k)
				return 1, nil
			})
	}
	if got[0] != idem.Key("ask-carrier-0", "run-1") || got[0] == got[1] {
		t.Fatalf("keys %v", got)
	}
}

// The crash window: the email went out, the journal
// write did not. Replay runs the step again with the
// same key; only a sender that honors it stays at one
// email per carrier.
func TestCrashAfterSideEffectBeforeJournal(t *testing.T) {
	for _, honors := range []bool{true, false} {
		j := newJournal()
		r := &Run{ID: "quote-78", DB: sqlfake.Open(j.h)}
		m := &mailer{honors: honors, sent: map[string]int{}}
		ctx := context.Background()
		j.failPut = 1 // crash between send and journal on step 0
		if err := quoteFlow(ctx, r, m, -1); err == nil {
			t.Fatal("want the journal failure")
		}
		if err := quoteFlow(ctx, r, m, -1); err != nil {
			t.Fatal(err)
		}
		want := 3
		if !honors {
			want = 4 // step 0's email went twice
		}
		if m.total != want {
			t.Fatalf("honors=%v emails=%d want %d", honors, m.total, want)
		}
	}
}

// Two workers replay one run with no claim: both run
// the step, the primary key rejects the second
// journal write, and the journal keeps the first.
func TestConcurrentReplayWithoutClaim(t *testing.T) {
	j := newJournal()
	db := sqlfake.Open(j.h)
	var runs atomic.Int64
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for w := range 2 {
		wg.Go(func() {
			_, errs[w] = Step(context.Background(), &Run{ID: "q-9", DB: db}, "ask",
				func(context.Context, string) (int, error) {
					runs.Add(1)
					<-start
					return w, nil
				})
		})
	}
	for runs.Load() < 2 {
		time.Sleep(time.Millisecond)
	}
	close(start)
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("want exactly one journal-write failure: %v", errs)
	}
	if runs.Load() != 2 || len(j.rows) != 1 {
		t.Fatalf("runs=%d rows=%d", runs.Load(), len(j.rows))
	}
}

// With a run claim, one worker replays at a time: the
// second gets ErrRunBusy and the step runs once.
func TestClaimOneWorkerPerRun(t *testing.T) {
	j := newJournal()
	db := sqlfake.Open(j.h)
	ctx := context.Background()
	m := &mailer{sent: map[string]int{}}
	var wg sync.WaitGroup
	var busy atomic.Int64
	for w := range 10 {
		wg.Go(func() {
			r := &Run{ID: "quote-80", DB: db}
			if err := r.Claim(ctx, fmt.Sprint("w", w), time.Minute); err != nil {
				if !errors.Is(err, ErrRunBusy) {
					t.Error(err)
				}
				busy.Add(1)
				return
			}
			if err := quoteFlow(ctx, r, m, -1); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if busy.Load() != 9 || m.total != 3 {
		t.Fatalf("busy=%d emails=%d", busy.Load(), m.total)
	}
	// The owner can renew; after expiry another worker takes over.
	var owner string
	for w := range 10 {
		if j.runs["quote-80"].owner == fmt.Sprint("w", w) {
			owner = fmt.Sprint("w", w)
		}
	}
	r := &Run{ID: "quote-80", DB: db}
	if err := r.Claim(ctx, owner, time.Minute); err != nil {
		t.Fatalf("renew: %v", err)
	}
	if err := r.Claim(ctx, "other", time.Minute); !errors.Is(err, ErrRunBusy) {
		t.Fatalf("steal: %v", err)
	}
	j.runs["quote-80"] = runRow{owner, time.Now().Add(-time.Second)}
	if err := r.Claim(ctx, "other", time.Minute); err != nil {
		t.Fatalf("takeover: %v", err)
	}
}
