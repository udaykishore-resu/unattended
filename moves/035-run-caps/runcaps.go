// Package runcaps puts three hard limits on an agent
// run: steps, tokens (all four Usage fields) and
// wall clock.
//
// Move 35: cap steps, tokens and wall clock per run.
package runcaps

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

// ErrBudget is a sentinel error.
var ErrBudget = errors.New("run budget spent")

// Limits are a run's hard caps.
type Limits struct {
	Steps, Tokens int
	Wall          time.Duration
}

// Run holds its Limits and two counters.
type Run struct {
	Limits        Limits
	steps, tokens int
}

func (r *Run) Charge(u llm.Usage) error {
	r.steps++
	r.tokens += u.InputTokens + u.CacheReadTokens +
		u.CacheWriteTokens + u.OutputTokens
	switch {
	case r.steps >= r.Limits.Steps:
		return fmt.Errorf("%w: %d steps",
			ErrBudget, r.steps)
	case r.tokens >= r.Limits.Tokens:
		return fmt.Errorf("%w: %d tokens",
			ErrBudget, r.tokens)
	}
	return nil
}

func WithWall(ctx context.Context, l Limits) (
	context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(ctx, l.Wall,
		fmt.Errorf("%w: wall clock", ErrBudget))
}
