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

func CrossCheck(inv Invoice) []string {
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
