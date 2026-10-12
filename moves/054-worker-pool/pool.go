// Package pool drains a queue with a fixed number of
// workers: the worker count is the concurrency limit.
//
// Move 54: queue plus bounded worker pool for batch
// jobs.
package pool

import (
	"context"
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
	handle func(context.Context, Job) error) {
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				if err := handle(ctx, j); err != nil {
					slog.Error("job failed",
						"id", j.ID, "err", err)
				}
			}
		}()
	}
	wg.Wait()
}
