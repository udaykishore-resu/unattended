package crosscheck

import (
	"math"
	"reflect"
	"testing"
)

func memphis() Invoice {
	return Invoice{
		Lines: []Line{
			{"linehaul", 3_800_00}, {"fuel", 308_00},
		},
		Subtotal: 4_108_00, Tax: 0, Total: 4_108_00,
	}
}

func TestCleanInvoicePasses(t *testing.T) {
	if p := CrossCheck(memphis()); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
}

// $4,180.00 against $4,108.00: two swapped digits in
// the total break the sum, so the invoice catches itself.
func TestSwappedDigitsCaught(t *testing.T) {
	inv := memphis()
	inv.Total = 4_180_00
	want := []string{"subtotal+tax 410800 != total 418000"}
	if p := CrossCheck(inv); !reflect.DeepEqual(p, want) {
		t.Fatalf("got %v", p)
	}
}

func TestSkippedLineCaught(t *testing.T) {
	inv := memphis()
	inv.Lines = inv.Lines[:1]
	if p := CrossCheck(inv); len(p) != 1 {
		t.Fatalf("got %v", p)
	}
}

// Reviewer cases. Discounts are negative lines on a
// normal invoice; the arithmetic handles them exactly.
func TestDiscountLine(t *testing.T) {
	inv := Invoice{
		Lines:    []Line{{"linehaul", 1_000_00}, {"volume discount", -50_00}},
		Subtotal: 950_00, Tax: 0, Total: 950_00,
	}
	if p := CrossCheck(inv); len(p) != 0 {
		t.Fatalf("discount: %v", p)
	}
}

func TestOneCentOffIsCaught(t *testing.T) {
	inv := memphis()
	inv.Total++
	if p := CrossCheck(inv); len(p) != 1 {
		t.Fatalf("one cent: %v", p)
	}
	inv = memphis()
	inv.Subtotal--
	inv.Total--
	if p := CrossCheck(inv); len(p) != 1 {
		t.Fatalf("one cent subtotal: %v", p)
	}
}

// A total the model did not find decodes as zero and
// fails; a missing total never passes.
func TestMissingTotalFails(t *testing.T) {
	inv := memphis()
	inv.Total = 0
	if p := CrossCheck(inv); len(p) == 0 {
		t.Fatal("missing total passed")
	}
}

// Overflow: values that would wrap int64 are refused
// before any sum is taken.
func TestOverflowRefused(t *testing.T) {
	huge := Invoice{
		Lines:    []Line{{"a", math.MaxInt64}, {"b", 1}},
		Subtotal: math.MinInt64, Total: math.MinInt64,
	}
	p := CrossCheck(huge)
	if len(p) != 1 || p[0] != "amounts or line count out of range" {
		t.Fatalf("got %v", p)
	}
	many := Invoice{Lines: make([]Line, maxLines+1)}
	if p := CrossCheck(many); len(p) != 1 {
		t.Fatalf("501 lines: %v", p)
	}
	edge := Invoice{
		Lines:    []Line{{"a", maxCents}},
		Subtotal: maxCents, Total: maxCents,
	}
	if p := CrossCheck(edge); len(p) != 0 {
		t.Fatalf("at the bound: %v", p)
	}
}

// Tax rounding: the tolerance comes from the carrier's
// rule, not a blanket number of cents.
func TestTaxRounding(t *testing.T) {
	// Three lines of $3.35 at 7.25%: per-line tax is
	// 24.2875 → 24 cents each, 72 in all. On the
	// subtotal ($10.05) it is 72.8625 → 73. One cent
	// apart, from rounding alone.
	inv := Invoice{
		Lines:    []Line{{"a", 335}, {"b", 335}, {"c", 335}},
		Subtotal: 1005, Tax: 72, Total: 1077,
	}
	if p := CrossCheck(inv); len(p) != 0 {
		t.Fatalf("sums: %v", p)
	}
	if p := TaxCheck(inv, 725, PerLine, 3); len(p) != 0 {
		t.Fatalf("per-line rounding within tolerance: %v", p)
	}
	if p := TaxCheck(inv, 725, PerInvoice, 3); len(p) != 1 {
		t.Fatalf("per-invoice rule must be exact: %v", p)
	}
	inv.Tax, inv.Total = 69, 1074 // 4 cents off: beyond 3 lines
	if p := TaxCheck(inv, 725, PerLine, 3); len(p) != 1 {
		t.Fatalf("beyond tolerance: %v", p)
	}
	neg := inv
	neg.Subtotal = -1005
	if p := TaxCheck(neg, 725, PerLine, 3); len(p) != 1 {
		t.Fatalf("credit subtotal: %v", p)
	}
	if p := TaxCheck(inv, -1, PerLine, 3); len(p) != 1 {
		t.Fatalf("negative rate: %v", p)
	}
}
