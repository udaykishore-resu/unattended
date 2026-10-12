package idem

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

// steps emulates the ai_steps table and its four
// queries (the SQL itself runs in Postgres in
// idem_integration_test.go).
type steps struct {
	mu         sync.Mutex
	rows       map[string]*row
	failFinish int // fail the next n finishQ calls
}

type row struct {
	lease time.Time
	owner string
	out   []byte
}

func (s *steps) h(q string, args []any) ([]string, [][]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := args[0].(string)
	r := s.rows[key]
	switch q {
	case claimQ:
		lease, tok := args[1].(time.Time), args[2].(string)
		if r == nil {
			s.rows[key] = &row{lease: lease, owner: tok}
			return nil, [][]any{{1}}, nil
		}
		if r.out == nil && r.lease.Before(time.Now()) {
			r.lease, r.owner = lease, tok // take over an expired lease
			return nil, [][]any{{1}}, nil
		}
		return nil, nil, nil
	case savedQ:
		return []string{"output"}, [][]any{{r.out}}, nil
	case releaseQ:
		if r.owner != args[1].(string) || r.out != nil {
			return nil, nil, nil
		}
		r.lease = time.Now().Add(-time.Second)
		return nil, [][]any{{1}}, nil
	case finishQ:
		if s.failFinish > 0 {
			s.failFinish--
			return nil, nil, errors.New("db: connection reset")
		}
		if r.out == nil {
			r.out = args[1].([]byte)
		}
		return []string{"output"}, [][]any{{r.out}}, nil
	}
	return nil, nil, errors.New("unknown query")
}

func newDB() *steps { return &steps{rows: map[string]*row{}} }

func (s *steps) expire(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[key].lease = time.Now().Add(-time.Second)
}

// The key is built from carrier and invoice number.
// A prompt fix (v7 -> v8) does not change it, so
// replaying failed runs never sends the email again.
func TestKeyStableAcrossPromptVersions(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	sends := 0
	step := func(promptVer string) func(context.Context, string) ([]byte, error) {
		return func(context.Context, string) ([]byte, error) {
			sends++ // the side effect: an email to the carrier
			return []byte("sent with " + promptVer), nil
		}
	}
	key := Key("dispute-email", "carrier:RDX", "invoice:40117")
	ctx := context.Background()

	out1, err := Once(ctx, db, key, step("v7"))
	if err != nil {
		t.Fatal(err)
	}
	// Same business identity after a prompt fix: same key.
	if Key("dispute-email", "carrier:RDX", "invoice:40117") != key {
		t.Fatal("key changed")
	}
	// Same key, different payload: the saved result
	// wins and the step does not run.
	out2, err := Once(ctx, db, key, step("v8"))
	if err != nil {
		t.Fatal(err)
	}
	if sends != 1 || string(out2) != string(out1) || string(out2) != "sent with v7" {
		t.Fatalf("sends=%d out1=%s out2=%s", sends, out1, out2)
	}
}

func TestLengthPrefix(t *testing.T) {
	if Key("s", "ab", "c") == Key("s", "a", "bc") {
		t.Fatal(`"ab"+"c" and "a"+"bc" must differ`)
	}
}

// The step receives the key, so it can pass it on.
func TestKeyReachesTheStep(t *testing.T) {
	db := sqlfake.Open(newDB().h)
	key := Key("file-ticket", "event:e-1")
	var got string
	Once(context.Background(), db, key, func(_ context.Context, k string) ([]byte, error) {
		got = k
		return []byte("T-1"), nil
	})
	if got != key {
		t.Fatalf("step saw %q", got)
	}
}

// A failed step releases its lease, so a retry runs.
func TestFailureReleases(t *testing.T) {
	db := sqlfake.Open(newDB().h)
	ctx := context.Background()
	key := Key("summarize", "event:e-9")
	if _, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
		return nil, errors.New("timeout")
	}); err == nil {
		t.Fatal("want the step error")
	}
	out, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
		return []byte("ok"), nil
	})
	if err != nil || string(out) != "ok" {
		t.Fatalf("retry: %s %v", out, err)
	}
}

// A second worker that finds a live lease gets ErrInFlight.
func TestInFlight(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	key := Key("x", "y")
	st.rows[key] = &row{lease: time.Now().Add(time.Hour), owner: "other"}
	_, err := Once(context.Background(), db, key, func(context.Context, string) ([]byte, error) {
		t.Fatal("must not run")
		return nil, nil
	})
	if !errors.Is(err, ErrInFlight) {
		t.Fatalf("got %v", err)
	}
}

// Fifty workers race for one key: the step runs once.
func TestConcurrentClaims(t *testing.T) {
	db := sqlfake.Open(newDB().h)
	key := Key("send-quote", "load:HF-1")
	var runs atomic.Int64
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_, err := Once(context.Background(), db, key,
				func(context.Context, string) ([]byte, error) {
					runs.Add(1)
					time.Sleep(5 * time.Millisecond)
					return []byte("sent"), nil
				})
			if err != nil && !errors.Is(err, ErrInFlight) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if runs.Load() != 1 {
		t.Fatalf("step ran %d times", runs.Load())
	}
}

// downstream is a ticket API. If it honors the key, a
// second request with the same key creates nothing.
type downstream struct {
	honors  bool
	created map[string]int
}

func (d *downstream) file(key string) {
	if d.honors && d.created[key] > 0 {
		return
	}
	d.created[key]++
}

// The crash window: the side effect happened, storing
// the output failed. Once says so, keeps the lease,
// and only the takeover after expiry reruns the step,
// with the same key.
func TestDoneButUnrecorded(t *testing.T) {
	for _, honors := range []bool{true, false} {
		st := newDB()
		db := sqlfake.Open(st.h)
		ctx := context.Background()
		api := &downstream{honors: honors, created: map[string]int{}}
		step := func(_ context.Context, k string) ([]byte, error) {
			api.file(k)
			return []byte("T-77"), nil
		}
		key := Key("file-ticket", "event:e-77")

		st.failFinish = 1
		out, err := Once(ctx, db, key, step)
		if !errors.Is(err, ErrUnrecorded) || string(out) != "T-77" {
			t.Fatalf("got %s %v", out, err)
		}
		// A retry before the lease expires does not rerun it.
		if _, err := Once(ctx, db, key, step); !errors.Is(err, ErrInFlight) {
			t.Fatalf("retry inside lease: %v", err)
		}
		st.expire(key)
		if out, err := Once(ctx, db, key, step); err != nil || string(out) != "T-77" {
			t.Fatalf("takeover: %s %v", out, err)
		}
		want := 1
		if !honors {
			want = 2 // the lease protects only our own row
		}
		if api.created[key] != want {
			t.Fatalf("honors=%v created=%d", honors, api.created[key])
		}
	}
}

// Worker A is still running when its lease expires
// and B takes over. A's failure must not release B's
// lease, or a third worker would start too.
func TestStaleHolderCannotReleaseTakeover(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	ctx := context.Background()
	key := Key("summarize", "event:e-5")

	aIn, aGo := make(chan struct{}), make(chan struct{})
	aDone := make(chan error)
	go func() {
		_, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
			close(aIn)
			<-aGo
			return nil, errors.New("A timed out")
		})
		aDone <- err
	}()
	<-aIn
	st.expire(key)

	bIn, bGo := make(chan struct{}), make(chan struct{})
	bDone := make(chan []byte)
	go func() {
		out, _ := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
			close(bIn)
			<-bGo
			return []byte("from B"), nil
		})
		bDone <- out
	}()
	<-bIn
	close(aGo)
	if err := <-aDone; err == nil {
		t.Fatal("A should fail")
	}
	// B still holds the key: C must not start.
	if _, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
		t.Fatal("C ran while B held the lease")
		return nil, nil
	}); !errors.Is(err, ErrInFlight) {
		t.Fatalf("C: %v", err)
	}
	close(bGo)
	if out := <-bDone; string(out) != "from B" {
		t.Fatalf("B: %s", out)
	}
}

// A stale holder and a takeover both finish: the
// first stored output wins and both callers get it.
func TestFirstFinishWins(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	ctx := context.Background()
	key := Key("draft", "invoice:9")

	aGo := make(chan struct{})
	aIn := make(chan struct{})
	aOut := make(chan []byte)
	go func() {
		out, _ := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
			close(aIn)
			<-aGo
			return []byte("A"), nil
		})
		aOut <- out
	}()
	<-aIn
	st.expire(key)
	outB, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
		return []byte("B"), nil
	})
	if err != nil || string(outB) != "B" {
		t.Fatalf("B: %s %v", outB, err)
	}
	close(aGo)
	if outA := <-aOut; string(outA) != "B" {
		t.Fatalf("A returned %s, want the stored B", outA)
	}
}

// A step that succeeds as its context ends is still
// recorded.
func TestRecordsAfterCancel(t *testing.T) {
	db := sqlfake.Open(newDB().h)
	ctx, cancel := context.WithCancel(context.Background())
	key := Key("send", "x")
	out, err := Once(ctx, db, key, func(context.Context, string) ([]byte, error) {
		cancel() // deadline hits right after the send
		return []byte("sent"), nil
	})
	if err != nil || string(out) != "sent" {
		t.Fatalf("%s %v", out, err)
	}
	if out, err := Once(context.Background(), db, key, nil); err != nil || string(out) != "sent" {
		t.Fatalf("saved: %s %v", out, err)
	}
}
