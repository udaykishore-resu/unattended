package abtest

import (
	"fmt"
	"testing"
)

func split(pct uint64, off *bool) Split {
	return Split{A: Arm{Name: "large"}, B: Arm{Name: "small"},
		Percent: pct, Salt: "exp-7",
		Disabled: func() bool { return *off }}
}

func TestByUnitNotByCall(t *testing.T) {
	off := false
	s := split(10, &off)
	b := 0
	for i := range 10000 {
		thread := fmt.Sprintf("ticket-thread-%d", i)
		first := s.For(thread).Name
		// Every call on one thread lands in the same arm.
		for range 3 {
			if s.For(thread).Name != first {
				t.Fatal("one thread got drafts from both models")
			}
		}
		if first == "small" {
			b++
		}
	}
	if b < 850 || b > 1150 {
		t.Fatalf("B got %d of 10000 units, want ~10%%", b)
	}
}

func TestKillSwitchSendsAllToA(t *testing.T) {
	off := true
	s := split(100, &off)
	for i := range 100 {
		if s.For(fmt.Sprint(i)).Name != "large" {
			t.Fatal("kill switch ignored")
		}
	}
	off = false
	if s.For("x").Name != "small" {
		t.Fatal("100% should all go to B once enabled")
	}
}

// Reuse a salt and the same shippers land in B
// every time; a new salt reshuffles them.
func TestNewSaltReshuffles(t *testing.T) {
	off := false
	a, b := split(50, &off), split(50, &off)
	b.Salt = "exp-8"
	same := 0
	for i := range 1000 {
		u := fmt.Sprintf("shipper-%d", i)
		if a.For(u).Name == b.For(u).Name {
			same++
		}
	}
	if same > 600 || same < 400 {
		t.Fatalf("salts not independent: %d of 1000 agree", same)
	}
}
