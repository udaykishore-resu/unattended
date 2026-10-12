package sampling

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

// A rare, costly label still reaches a person. At
// Halcyon ops@ gets ~900 emails a day and ~2% are
// damage claims: 18 a day. A flat 1% would review one
// every few days; the stratum rate reviews ~3.6 a day.
func TestRareCategoryIsSampled(t *testing.T) {
	const days, perDay = 30, 18
	picked := 0
	for i := range days * perDay {
		if Pick(fmt.Sprintf("claim-%d", i), "damage_claim", "2026-q4") {
			picked++
		}
	}
	if picked < 80 || picked > 140 { // ~108 expected
		t.Fatalf("damage claims sampled in a month: %d", picked)
	}
}

func TestPlantedItemRefused(t *testing.T) {
	sent, counted := 0, 0
	do := func() error { sent++; return nil }
	count := func(string) { counted++ }
	if err := Execute(Item{ID: "seed-1", Planted: true}, do, count); !errors.Is(err, ErrPlanted) {
		t.Fatalf("err = %v", err)
	}
	if err := Execute(Item{ID: "real-1"}, do, count); err != nil {
		t.Fatal(err)
	}
	if sent != 1 || counted != 1 {
		t.Fatalf("sent=%d counted=%d; the planted item leaked", sent, counted)
	}
}

func TestWeightedByVolume(t *testing.T) {
	// High sample error on a rare label barely moves the
	// overall rate; the per-stratum rate still shows it.
	rate := map[string]float64{"damage_claim": 0.10, "other": 0.02}
	vol := map[string]int{"damage_claim": 20, "other": 880}
	got := Weighted(rate, vol)
	if math.Abs(got-(0.10*20+0.02*880)/900) > 1e-12 {
		t.Fatalf("weighted = %v", got)
	}
	if Weighted(rate, nil) != 0 {
		t.Fatal("no volume")
	}
}
