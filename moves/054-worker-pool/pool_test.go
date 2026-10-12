package pool

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// 2,400 invoices, eight workers: never more than
// eight requests in flight.
func TestBoundedConcurrency(t *testing.T) {
	jobs := make(chan Job)
	go func() {
		defer close(jobs) // the producer closes jobs
		for i := range 2400 {
			jobs <- Job{ID: fmt.Sprintf("inv-%d", i)}
		}
	}()
	var inFlight, peak, done atomic.Int64
	err := RunPool(context.Background(), jobs, 8, func(context.Context, Job) error {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(50 * time.Microsecond)
		inFlight.Add(-1)
		done.Add(1)
		return nil
	})
	if err != nil || done.Load() != 2400 || peak.Load() > 8 {
		t.Fatalf("done=%d peak=%d", done.Load(), peak.Load())
	}
}

func TestCancelStopsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	jobs := make(chan Job, 100)
	for i := range 100 {
		jobs <- Job{ID: fmt.Sprint(i)}
	}
	close(jobs)
	var done atomic.Int64
	err := RunPool(ctx, jobs, 1, func(context.Context, Job) error {
		if done.Add(1) == 10 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || done.Load() != 10 {
		t.Fatalf("handled %d after cancel, err=%v", done.Load(), err)
	}
	// The other 90 are still in the channel: abandoned
	// unless the caller requeues them from the source.
	if len(jobs) != 90 {
		t.Fatalf("left in channel = %d", len(jobs))
	}
}

func TestFailuresAreLoggedNotFatal(t *testing.T) {
	jobs := make(chan Job, 3)
	jobs <- Job{ID: "a"}
	jobs <- Job{ID: "poison"}
	jobs <- Job{ID: "c"}
	close(jobs)
	var done atomic.Int64
	err := RunPool(context.Background(), jobs, 2, func(_ context.Context, j Job) error {
		done.Add(1)
		if j.ID == "poison" {
			return fmt.Errorf("malformed PDF")
		}
		return nil
	})
	if err != nil || done.Load() != 3 {
		t.Fatalf("done = %d err=%v", done.Load(), err)
	}
}

// Zero or negative workers would start nothing and
// leave the producer blocked: refuse before starting.
func TestWorkerCountValidated(t *testing.T) {
	for _, n := range []int{0, -1} {
		jobs := make(chan Job, 1)
		jobs <- Job{ID: "a"}
		close(jobs)
		called := false
		err := RunPool(context.Background(), jobs, n, func(context.Context, Job) error {
			called = true
			return nil
		})
		if err == nil || called {
			t.Fatalf("workers=%d: err=%v called=%v", n, err, called)
		}
	}
}

// Cancel while a worker is mid-item: RunPool waits for
// that handler to return (it sees ctx), starts no new
// item, and reports the cancellation.
func TestCancelDuringItem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	jobs := make(chan Job, 10)
	for i := range 10 {
		jobs <- Job{ID: fmt.Sprint(i)}
	}
	close(jobs)
	started := make(chan struct{}, 10)
	var finished atomic.Int64
	go func() { <-started; cancel() }()
	err := RunPool(ctx, jobs, 2, func(ctx context.Context, j Job) error {
		started <- struct{}{}
		<-ctx.Done() // a long model call that honors ctx
		finished.Add(1)
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if n := finished.Load(); n < 1 || n > 2 {
		t.Fatalf("in-flight items finished = %d", n)
	}
	if left := len(jobs); left != 10-int(finished.Load()) {
		t.Fatalf("left = %d", left)
	}
}

// Every item failing does not stop the pool: each is
// tried once and logged; retries and the dead-letter
// table are Move 56's job, not the pool's.
func TestEveryItemFails(t *testing.T) {
	jobs := make(chan Job, 50)
	for i := range 50 {
		jobs <- Job{ID: fmt.Sprint(i)}
	}
	close(jobs)
	var calls atomic.Int64
	err := RunPool(context.Background(), jobs, 4, func(context.Context, Job) error {
		calls.Add(1)
		return fmt.Errorf("malformed PDF")
	})
	if err != nil || calls.Load() != 50 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestEstimateIsBytesOverFour(t *testing.T) {
	if EstimateTokens("") != 0 || EstimateTokens("abcd") != 1 || EstimateTokens("abcde") != 2 {
		t.Fatal("estimate")
	}
}
