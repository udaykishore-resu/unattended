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
	if tries < 1 || base <= 0 || ceil < base {
		return ErrPolicy
	}
	var err error
	for i := range tries {
		if err = fn(ctx); err == nil {
			return nil
		}
		var p Permanent
		if errors.As(err, &p) || i == tries-1 {
			return err
		}
		d := ceil // base<<i, capped before it overflows
		if base <= ceil>>i {
			d = base << i
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rand.N(d)):
		}
	}
	return err
}

// ErrPolicy: tries below 1, a non-positive base, or a
// ceiling below the base. Refused before any attempt,
// so a zero-value policy can't report success without
// calling fn, and rand.N never sees a bad duration.
var ErrPolicy = errors.New("retry: bad policy")

// Permanent is a small error type that wraps errors
// no retry can fix: a 400, a schema the model failed
// twice, a missing customer.
type Permanent struct{ Err error }

func (p Permanent) Error() string { return "permanent: " + p.Err.Error() }
func (p Permanent) Unwrap() error { return p.Err }

// DeadLetter is the DLQ row when retries run out.
// Owner is a person, not a team alias. InputRef
// points at the input (an invoice ID), so the row
// holds enough to replay without copying invoice
// text into the DLQ. Described in prose; minimal.
type DeadLetter struct {
	Workflow, Key, InputRef, LastErr string
	Tries                            int
	Owner                            string
	Replay                           string // command
}
