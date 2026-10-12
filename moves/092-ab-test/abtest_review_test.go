package abtest

import (
	"fmt"
	"testing"

	drift "github.com/udaykishore-resu/unattended/moves/090-drift-alarms"
)

// Both arms see comparable traffic: assign 20,000
// tickets of four types and compare the type mix per
// arm with Move 90's PSI. A fair split stays far
// under the 0.1 "look" line.
func TestArmsSeeComparableMix(t *testing.T) {
	off := false
	s := split(50, &off)
	types := []string{"wismo", "detention", "damage_claim", "rate_request"}
	mix := map[string]map[string]int{"large": {}, "small": {}}
	for i := range 20000 {
		ty := types[i%7%4] // uneven mix
		mix[s.For(fmt.Sprintf("thread-%d", i)).Name][ty]++
	}
	v, ok := drift.PSI(mix["large"], mix["small"], 1000)
	if !ok || v > 0.01 {
		t.Fatalf("arm mix PSI = %.4f (ok=%v)", v, ok)
	}
}

// Salt and unit are joined with a separator, so
// ("exp-7", "1x") and ("exp-71", "x") hash apart.
func TestSaltSeparator(t *testing.T) {
	a := Split{A: Arm{Name: "a"}, B: Arm{Name: "b"}, Percent: 50, Salt: "exp-7"}
	b := a
	b.Salt = "exp-71"
	differ := 0
	for i := range 200 {
		u := fmt.Sprint(i)
		if a.For("1"+u).Name != b.For(u).Name {
			differ++
		}
	}
	if differ == 0 {
		t.Fatal("salt and unit collide")
	}
}

func TestDiffCI(t *testing.T) {
	// Hundreds per arm: a 3-point gain is noise.
	lo, hi := DiffCI(240, 400, 252, 400)
	if !(lo < 0 && hi > 0) {
		t.Fatalf("400/arm: [%.3f, %.3f] should span zero", lo, hi)
	}
	// Thousands per arm: the same gain is resolved.
	lo, hi = DiffCI(3000, 5000, 3150, 5000)
	if !(lo > 0 && hi < 0.06) {
		t.Fatalf("5000/arm: [%.3f, %.3f] should exclude zero", lo, hi)
	}
}
