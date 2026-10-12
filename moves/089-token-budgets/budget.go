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

	"github.com/udaykishore-resu/unattended/llm"
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
	if n := Total(resp.Usage); n > 0 || err == nil {
		got = n
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
// case won't fit.
var ErrBudget = errors.New("token budget spent")

// Ledger is a mutex and two maps, used and limit.
type Ledger struct {
	mu          sync.Mutex
	used, limit map[string]int
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
		return fmt.Errorf("%w: %s", ErrBudget, k)
	}
	l.used[k] += n
	return nil
}

// Settle swaps the reservation for the real spend.
func (l *Ledger) Settle(k string, reserved, got int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.used[k] += got - reserved
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
