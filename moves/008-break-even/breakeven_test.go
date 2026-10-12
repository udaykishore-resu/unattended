package breakeven

import (
	"math"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// The Halcyon numbers from the book: 9,000 in, 400
// out, 1.2 attempts, 60% kept => 2 attempts per kept.
func TestRejectedRunsDoubleTheBill(t *testing.T) {
	r := Ratios{Read: 0.1, Write: 1.25, Out: 5}
	per := Equiv(llm.Usage{InputTokens: 9000, OutputTokens: 400}, r)
	if !near(per, 9000+400*5) {
		t.Fatalf("per try = %v", per)
	}
	if got := PerKept(per, 1.2, 0.6); !near(got, 2*per) {
		t.Fatalf("per kept = %v, want 2 attempts' worth (%v)", got, 2*per)
	}
	// Raising acceptance from 60 to 80 percent cuts it by a quarter.
	if got := PerKept(per, 1.2, 0.8); !near(got, 0.75*PerKept(per, 1.2, 0.6)) {
		t.Fatalf("80%% kept = %v", got)
	}
}

func TestEquivCountsAllFourFields(t *testing.T) {
	u := llm.Usage{InputTokens: 100, CacheReadTokens: 1000,
		CacheWriteTokens: 200, OutputTokens: 10}
	got := Equiv(u, Ratios{Read: 0.1, Write: 1.25, Out: 4})
	if !near(got, 100+100+250+40) {
		t.Fatalf("got %v", got)
	}
}

func TestBudgetSubtractsReview(t *testing.T) {
	// 18 minutes saved, 4 of review: 14 count.
	got := Budget(18, 4, 1, 0.5)
	if !near(got, 28) {
		t.Fatalf("got %v", got)
	}
}
