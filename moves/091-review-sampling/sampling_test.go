package sampling

import (
	"fmt"
	"math"
	"testing"
)

func rate(stratum string, n int) float64 {
	k := 0
	for i := range n {
		if Pick(fmt.Sprintf("email-%d", i), stratum, "2026-q4") {
			k++
		}
	}
	return float64(k) / float64(n)
}

func TestRatesPerStratum(t *testing.T) {
	const n = 20000
	for stratum, want := range map[string]float64{
		"damage_claim": 0.20, "rate_request": 0.03, "other": 0.15,
		"hazmat": 0.10, // unknown label: look hard
	} {
		if got := rate(stratum, n); math.Abs(got-want) > 0.015 {
			t.Errorf("%s: sampled %.3f, want ~%.2f", stratum, got, want)
		}
	}
}

// A redelivery isn't sampled twice or skipped by luck.
func TestDeterministic(t *testing.T) {
	for i := range 100 {
		id := fmt.Sprintf("e-%d", i)
		if Pick(id, "other", "s") != Pick(id, "other", "s") {
			t.Fatal("same id, different answer")
		}
	}
}

func TestFloor(t *testing.T) {
	Rates["newsletter"] = 0
	defer delete(Rates, "newsletter")
	if got := rate("newsletter", 20000); got < 0.005 {
		t.Fatalf("floor not applied: %.4f", got)
	}
}
