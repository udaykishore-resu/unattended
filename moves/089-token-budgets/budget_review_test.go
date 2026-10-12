package budget

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
	retry "github.com/udaykishore-resu/unattended/moves/056-retry-dlq"
)

// A success that reports zero usage settles the
// worst case: zero means the adapter lost the count.
func TestZeroUsageOnSuccessKeepsWorst(t *testing.T) {
	l := NewLedger(map[string]int{"triage/acme": 100_000})
	m := budgeted(&flaky{}, l) // answers with empty Usage
	if _, err := m.Complete(context.Background(), req()); err != nil {
		t.Fatal(err)
	}
	if got := l.Used("triage/acme"); got != worst {
		t.Fatalf("used = %d, want worst case %d", got, worst)
	}
}

// An error that still reports usage is charged what
// it reports.
type partial struct{}

func (partial) Complete(context.Context, llm.Request) (llm.Response, error) {
	return llm.Response{Usage: llm.Usage{InputTokens: 70}}, errors.New("stream cut")
}

func TestErrorWithUsageChargesUsage(t *testing.T) {
	l := NewLedger(map[string]int{"triage/acme": 100_000})
	if _, err := budgeted(partial{}, l).Complete(context.Background(), req()); err == nil {
		t.Fatal("want error")
	}
	if got := l.Used("triage/acme"); got != 70 {
		t.Fatalf("used = %d", got)
	}
}

// The refusal is Permanent by construction: the
// retry layer tries once, and ErrBudget still matches.
func TestRefusalIsPermanent(t *testing.T) {
	m := budgeted(&flaky{}, NewLedger(map[string]int{"triage/acme": 1}))
	tries := 0
	err := retry.Retry(context.Background(), 3, time.Millisecond, time.Millisecond,
		func(ctx context.Context) error {
			tries++
			_, err := m.Complete(ctx, req())
			return err
		})
	var p retry.Permanent
	if tries != 1 || !errors.Is(err, ErrBudget) || !errors.As(err, &p) {
		t.Fatalf("tries=%d err=%v", tries, err)
	}
}

// The day rolls in the business time zone. At 19:30
// in Chicago it is already tomorrow in UTC; a UTC
// reset would land in the evening rush.
func TestRollOnBusinessDay(t *testing.T) {
	chicago := time.FixedZone("CDT", -5*3600)
	l := NewLedger(map[string]int{"triage/acme": 10 * worst})
	evening := time.Date(2026, 10, 12, 19, 30, 0, 0, chicago)
	if l.Roll(evening, chicago) {
		t.Fatal("first Roll only stamps the day")
	}
	l.Reserve("triage/acme", worst)
	// 01:30 UTC on the 13th: still the 12th in Chicago.
	if l.Roll(evening.Add(time.Hour).UTC(), chicago) {
		t.Fatal("reset in the business evening")
	}
	if l.Used("triage/acme") != worst {
		t.Fatal("counters lost mid-day")
	}
	// Midnight in Chicago.
	if !l.Roll(time.Date(2026, 10, 13, 0, 1, 0, 0, chicago), chicago) {
		t.Fatal("no reset at the business midnight")
	}
	if l.Used("triage/acme") != 0 {
		t.Fatal("counters not cleared")
	}
	// A reservation from yesterday settling now can't
	// drive today's counter negative.
	l.Settle("triage/acme", worst, 10)
	if got := l.Used("triage/acme"); got != 0 {
		t.Fatalf("used = %d", got)
	}
}

// The documented limit: a process-local ledger does
// not share a tenant budget across pods. Two replicas
// each allow the full limit, so the tenant spends
// twice it. A shared store (Postgres, Redis) fixes it.
func TestReplicasEachGetAFullBudget(t *testing.T) {
	limits := map[string]int{"triage/acme": 3 * worst}
	podA, podB := NewLedger(limits), NewLedger(limits)
	f := &flaky{u: llm.Usage{InputTokens: worst}}
	ok := 0
	for _, l := range []*Ledger{podA, podB} {
		m := budgeted(f, l)
		for range 5 {
			if _, err := m.Complete(context.Background(), req()); err == nil {
				ok++
			}
		}
	}
	if ok != 6 {
		t.Fatalf("ok = %d; want 2 pods x 3", ok)
	}
}
