package crosscheck

import (
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
