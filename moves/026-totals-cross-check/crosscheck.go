// Package crosscheck makes an extracted invoice
// check its own arithmetic.
//
// Move 26: invoice extraction with a totals
// cross-check.
package crosscheck

import "fmt"

// Invoice holds every printed number in integer
// cents. Described in prose; minimal.
type Invoice struct {
	Lines                []Line
	Subtotal, Tax, Total int64
}

type Line struct {
	Desc        string
	AmountCents int64
}

// Halcyon's bounds: at most 500 lines, every amount
// within ±$10 million. With them, no sum below can
// overflow int64 (500 × 1e9 is far below 9.2e18).
const (
	maxLines = 500
	maxCents = 10_000_000_00
)

func in(c int64) bool {
	return c >= -maxCents && c <= maxCents
}

// bounded rejects invoices whose numbers are outside
// the bounds, so the sums in CrossCheck stay exact.
func bounded(inv Invoice) bool {
	if len(inv.Lines) > maxLines || !in(inv.Subtotal) ||
		!in(inv.Tax) || !in(inv.Total) {
		return false
	}
	for _, l := range inv.Lines {
		if !in(l.AmountCents) {
			return false
		}
	}
	return true
}

func CrossCheck(inv Invoice) []string {
	if !bounded(inv) {
		return []string{"amounts or line count " +
			"out of range"}
	}
	var p []string
	var sum int64
	for _, l := range inv.Lines {
		sum += l.AmountCents
	}
	if sum != inv.Subtotal {
		p = append(p, fmt.Sprintf(
			"lines sum %d != subtotal %d",
			sum, inv.Subtotal))
	}
	if inv.Subtotal+inv.Tax != inv.Total {
		p = append(p, fmt.Sprintf(
			"subtotal+tax %d != total %d",
			inv.Subtotal+inv.Tax, inv.Total))
	}
	return p
}

// Rounding is the carrier's documented tax-rounding
// rule. It sets the only tolerance in this package.
type Rounding int

const (
	// PerInvoice: tax is computed once on the subtotal
	// and rounded half up. Tolerance 0.
	PerInvoice Rounding = iota
	// PerLine: tax is rounded on each taxed line. Each
	// rounding moves the sum by at most half a cent,
	// so the tolerance is one cent per taxed line.
	PerLine
)

// TaxCheck compares printed tax with subtotal × rate
// (rate in basis points). It is the only check with a
// tolerance; the sums above stay exact. Credits have
// their own document type, so a negative subtotal is
// refused here.
func TaxCheck(inv Invoice, rateBP int64, r Rounding,
	taxedLines int) []string {
	if !bounded(inv) || inv.Subtotal < 0 ||
		rateBP < 0 || rateBP > 10_000 || taxedLines < 0 {
		return []string{"tax inputs out of range"}
	}
	want := (inv.Subtotal*rateBP + 5_000) / 10_000
	tol := int64(0)
	if r == PerLine {
		tol = int64(taxedLines)
	}
	d := inv.Tax - want
	if d < -tol || d > tol {
		return []string{fmt.Sprintf(
			"tax %d != %d (±%d)", inv.Tax, want, tol)}
	}
	return nil
}
