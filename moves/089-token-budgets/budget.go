// Package budget reserves a worst-case token cost
// before every call and settles the real number
// after, per workflow and tenant.
//
// Move 89: token budgets per workflow and per tenant.
package budget

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
)

func (b Budgeted) Complete(ctx context.Context,
	r llm.Request) (llm.Response, error) {
	k := b.Key(ctx) // workflow and tenant
	worst := b.EstIn(r) + r.MaxTokens
	if err := b.Ledger.Reserve(k, worst); err != nil {
		return llm.Response{}, err
	}
	got := worst // panic or no usage: assume spent
	defer func() { b.Ledger.Settle(k, worst, got) }()
	resp, err := b.Next.Complete(ctx, r)
	if n := Total(resp.Usage); n > 0 {
		got = n // zero is "unknown", not "free"
	}
	return resp, err
}

// Total counts all four fields: cache writes
// and reads are spend too.
func Total(u llm.Usage) int {
	return u.InputTokens + u.CacheReadTokens +
		u.CacheWriteTokens + u.OutputTokens
}

// Budgeted is an llm.Model wrapper. It sits inside
// the retry layer, so every attempt is charged.
type Budgeted struct {
	Next   llm.Model
	Ledger *Ledger
	// Key names the budget: workflow and tenant,
	// read from the context.
	Key func(context.Context) string
	// EstIn estimates input tokens; bytes / 4 is fine.
	EstIn func(llm.Request) int
}

// ErrBudget: the key has no limit, or the worst
// case won't fit. Reserve returns it wrapped in
// Move 56's Permanent, so the retry layer stops at
// once; errors.Is(err, ErrBudget) still holds.
var ErrBudget = errors.New("token budget spent")

// Ledger is a mutex and two maps, used and limit,
// plus the business day the counters belong to.
// It guards one process only: each replica holds its
// own counters (see TestReplicasEachGetAFullBudget).
type Ledger struct {
	mu          sync.Mutex
	used, limit map[string]int
	day         string
}

func NewLedger(limits map[string]int) *Ledger {
	return &Ledger{used: map[string]int{},
		limit: limits}
}

// Reserve refuses when the key has no limit or the
// worst case won't fit, so a new workflow needs a
// budget before it runs.
func (l *Ledger) Reserve(k string, n int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limit[k]
	if !ok || l.used[k]+n > lim {
		return retry.Permanent{
			Err: fmt.Errorf("%w: %s", ErrBudget, k)}
	}
	l.used[k] += n
	return nil
}

// Settle swaps the reservation for the real spend.
func (l *Ledger) Settle(k string, reserved, got int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Floor at zero: a reservation made before Roll
	// settles after it.
	l.used[k] = max(0, l.used[k]+got-reserved)
}

// Roll zeroes the counters when the business day
// changes. Call it before Reserve with the current
// time; loc is the business time zone from config,
// not UTC. It returns true when it reset.
func (l *Ledger) Roll(now time.Time, loc *time.Location) bool {
	day := now.In(loc).Format(time.DateOnly)
	l.mu.Lock()
	defer l.mu.Unlock()
	if day == l.day {
		return false
	}
	reset := l.day != ""
	l.day = day
	if reset {
		clear(l.used)
	}
	return reset
}

// Used reports spend so far for a key.
func (l *Ledger) Used(k string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.used[k]
}

// EstBytes is the bytes/4 input estimate.
func EstBytes(r llm.Request) int {
	n := len(r.System)
	for _, m := range r.Messages {
		n += len(m.Content)
	}
	for _, t := range r.Tools {
		n += len(t.Name) + len(t.Description) +
			len(t.InputSchema)
	}
	return n/4 + 1
}
