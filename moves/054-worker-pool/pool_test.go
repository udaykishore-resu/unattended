package pool

import (
	"context"
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
	RunPool(context.Background(), jobs, 8, func(context.Context, Job) error {
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
	if done.Load() != 2400 || peak.Load() > 8 {
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
	RunPool(ctx, jobs, 1, func(context.Context, Job) error {
		if done.Add(1) == 10 {
			cancel()
		}
		return nil
	})
	if done.Load() != 10 {
		t.Fatalf("handled %d after cancel", done.Load())
	}
}

func TestFailuresAreLoggedNotFatal(t *testing.T) {
	jobs := make(chan Job, 3)
	jobs <- Job{ID: "a"}
	jobs <- Job{ID: "poison"}
	jobs <- Job{ID: "c"}
	close(jobs)
	var done atomic.Int64
	RunPool(context.Background(), jobs, 2, func(_ context.Context, j Job) error {
		done.Add(1)
		if j.ID == "poison" {
			return fmt.Errorf("malformed PDF")
		}
		return nil
	})
	if done.Load() != 3 {
		t.Fatalf("done = %d", done.Load())
	}
}
