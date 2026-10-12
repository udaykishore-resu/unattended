// Package retry retries with exponential backoff and
// full jitter, and stops at once on a permanent
// error.
//
// Move 56: retries with jitter and a dead-letter
// queue humans read.
package retry

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

func Retry(ctx context.Context, tries int,
	base, ceil time.Duration,
	fn func(context.Context) error) error {
	var err error
	for i := range tries {
		if err = fn(ctx); err == nil {
			return nil
		}
		var p Permanent
		if errors.As(err, &p) {
			return err
		}
		sleep := rand.N(min(ceil, base<<i))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
	}
	return err
}

// Permanent is a small error type that wraps errors
// no retry can fix: a 400, a schema the model failed
// twice, a missing customer.
type Permanent struct{ Err error }

func (p Permanent) Error() string { return "permanent: " + p.Err.Error() }
func (p Permanent) Unwrap() error { return p.Err }

// DeadLetter is the DLQ row when retries run out.
// Owner is a person, not a team alias. Described in
// prose; minimal.
type DeadLetter struct {
	Workflow, Key, Input, LastErr string
	Tries                         int
	Owner                         string
	Replay                        string // command
}
