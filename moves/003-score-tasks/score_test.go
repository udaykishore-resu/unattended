package scoretasks

import (
	"slices"
	"testing"
)

func TestMultiplySinksIrreversible(t *testing.T) {
	// Huge volume and pain, but undo is 1.
	payments := Task{Name: "payments", PerWeek: 200,
		Minutes: 6, Pain: 5, Undo: 1}
	tagging := Task{Name: "ops@ tagging", PerWeek: 4500,
		Minutes: 20.0 / 60, Pain: 3, Undo: 5}

	if Score(tagging) <= Score(payments) {
		t.Fatalf("tagging %.0f should beat payments %.0f",
			Score(tagging), Score(payments))
	}
	if FirstProject(payments) {
		t.Error("payments (undo 1) must not be a first project")
	}
	if !FirstProject(tagging) {
		t.Error("tagging should qualify")
	}
}

func TestRareTaskDropped(t *testing.T) {
	rare := Task{PerWeek: 2, Minutes: 120, Pain: 5, Undo: 5}
	if FirstProject(rare) {
		t.Error("a twice-a-week task should not be a first project")
	}
}

func TestRanking(t *testing.T) {
	ts := []Task{
		{Name: "a", PerWeek: 10, Minutes: 6, Pain: 2, Undo: 3},
		{Name: "b", PerWeek: 10, Minutes: 6, Pain: 4, Undo: 3},
	}
	slices.SortFunc(ts, func(x, y Task) int {
		return int(Score(y) - Score(x))
	})
	if ts[0].Name != "b" {
		t.Fatalf("want b first, got %s", ts[0].Name)
	}
}
