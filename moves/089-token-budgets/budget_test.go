package budget

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
)

// flaky fails its first n calls with no usage, then
// answers with u.
type flaky struct {
	mu    sync.Mutex
	fails int
	u     llm.Usage
	calls int
}

func (f *flaky) Complete(context.Context, llm.Request) (llm.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.fails {
		return llm.Response{}, errors.New("timeout")
	}
	return llm.Response{Text: "ok", Usage: f.u, StopReason: llm.StopEnd}, nil
}

func key(context.Context) string { return "triage/acme" }

const worst = 100 + 500 // EstIn + MaxTokens

func req() llm.Request { return llm.Request{MaxTokens: 500} }

func budgeted(next llm.Model, l *Ledger) Budgeted {
	return Budgeted{Next: next, Ledger: l, Key: key,
		EstIn: func(llm.Request) int { return 100 }}
}

// Retry outside, budget inside: every attempt pays.
// Two timeouts with no usage are charged the worst
// case; the success is charged its real four fields.
func TestBudgetChargesEveryRetry(t *testing.T) {
	l := NewLedger(map[string]int{"triage/acme": 100_000})
	f := &flaky{fails: 2, u: llm.Usage{InputTokens: 40, CacheReadTokens: 300,
		CacheWriteTokens: 20, OutputTokens: 60}}
	m := budgeted(f, l)
	err := retry.Retry(context.Background(), 5, time.Millisecond, 2*time.Millisecond,
		func(ctx context.Context) error {
			_, err := m.Complete(ctx, req())
			return err
		})
	if err != nil {
		t.Fatal(err)
	}
	want := 2*worst + (40 + 300 + 20 + 60)
	if got := l.Used("triage/acme"); f.calls != 3 || got != want {
		t.Fatalf("calls=%d used=%d, want 3 calls and %d", f.calls, got, want)
	}
}

// A spent budget fails fast, and wrapped as
// Permanent it doesn't turn into five refusals.
func TestRunawayStopsAtTheLimit(t *testing.T) {
	l := NewLedger(map[string]int{"triage/acme": 3 * worst})
	f := &flaky{u: llm.Usage{InputTokens: worst}} // spends its worst case
	m := budgeted(f, l)
	ok := 0
	for range 10 {
		if _, err := m.Complete(context.Background(), req()); err == nil {
			ok++
		} else if !errors.Is(err, ErrBudget) {
			t.Fatal(err)
		}
	}
	if ok != 3 || f.calls != 3 {
		t.Fatalf("ok=%d calls=%d; the 4th call must never reach the model", ok, f.calls)
	}
	tries := 0
	retry.Retry(context.Background(), 5, time.Millisecond, time.Millisecond,
		func(ctx context.Context) error {
			tries++
			if _, err := m.Complete(ctx, req()); errors.Is(err, ErrBudget) {
				return retry.Permanent{Err: err}
			}
			return nil
		})
	if tries != 1 {
		t.Fatalf("budget refusal retried %d times", tries)
	}
}

func TestNoBudgetNoRun(t *testing.T) {
	m := budgeted(&flaky{}, NewLedger(nil))
	if _, err := m.Complete(context.Background(), req()); !errors.Is(err, ErrBudget) {
		t.Fatalf("got %v", err)
	}
}

// Fifty workers check together; reservations keep
// them inside the limit.
func TestConcurrentReserve(t *testing.T) {
	l := NewLedger(map[string]int{"triage/acme": 10 * worst})
	f := &flaky{u: llm.Usage{InputTokens: worst}}
	m := budgeted(f, l)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { m.Complete(context.Background(), req()) })
	}
	wg.Wait()
	if f.calls != 10 || l.Used("triage/acme") != 10*worst {
		t.Fatalf("calls=%d used=%d", f.calls, l.Used("triage/acme"))
	}
}

// A panic in the next layer can't leak the reservation.
func TestPanicSettlesWorstCase(t *testing.T) {
	l := NewLedger(map[string]int{"k": 10_000})
	m := Budgeted{Next: panicky{}, Ledger: l,
		Key: func(context.Context) string { return "k" }, EstIn: EstBytes}
	func() {
		defer func() { recover() }()
		m.Complete(context.Background(), llm.Request{System: "abcd", MaxTokens: 10})
	}()
	if got := l.Used("k"); got != 2+10 {
		t.Fatalf("used = %d", got)
	}
}

type panicky struct{}

func (panicky) Complete(context.Context, llm.Request) (llm.Response, error) {
	panic("adapter bug")
}
