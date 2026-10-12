// Package pool drains a queue with a fixed number of
// workers: the worker count is the concurrency limit.
//
// The queue here is a channel, so it lives in memory:
// on cancellation each worker finishes (or fails
// fast) the job it holds and takes no more; jobs
// still in the channel are abandoned. Jobs that must
// survive a restart need a table queue with items
// marked done as they finish.
//
// Move 54: queue plus bounded worker pool for batch
// jobs.
package pool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Job is any struct with an ID.
type Job struct {
	ID    string
	Input string
}

func RunPool(ctx context.Context, jobs <-chan Job,
	workers int,
	handle func(context.Context, Job) error) error {
	if workers < 1 {
		return fmt.Errorf("pool: %d workers", workers)
	}
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for ctx.Err() == nil {
				j, ok := <-jobs
				if !ok {
					return
				}
				if err := handle(ctx, j); err != nil {
					slog.Error("job failed",
						"id", j.ID, "err", err)
				}
			}
		})
	}
	wg.Wait()
	return ctx.Err() // non-nil: jobs may be left
}

// EstimateTokens is the rough reservation each worker
// takes from the shared token budget before a call:
// bytes divided by four. It is a heuristic, not a
// count; reconcile the reservation against the
// llm.Usage the call reports (Move 89).
func EstimateTokens(input string) int {
	return (len(input) + 3) / 4
}
