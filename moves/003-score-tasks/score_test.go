package scoretasks

import (
	"math"
	"slices"
	"testing"
)

// The rows the book prints for Halcyon. Tagging:
// 900 emails a day, five days, 20 s each = 25 h.
var (
	tagging = Task{Name: "ops@ tagging", PerWeek: 4500,
		Minutes: 20.0 / 60, Pain: 3, Undo: 5}
	payments = Task{Name: "payments", PerWeek: 200,
		Minutes: 6, Pain: 4, Undo: 1}
	checkCalls = Task{Name: "check calls", PerWeek: 280,
		Minutes: 0.75, Pain: 4, Undo: 3}
)

func TestHalcyonScores(t *testing.T) {
	for _, c := range []struct {
		t     Task
		hours float64
		want  float64
	}{
		{tagging, 25, 375},
		{payments, 20, 80},
		{checkCalls, 3.5, 42},
	} {
		if err := c.t.Valid(); err != nil {
			t.Fatal(err)
		}
		if h := c.t.PerWeek * c.t.Minutes / 60; math.Abs(h-c.hours) > 1e-9 {
			t.Errorf("%s: %v h/wk, want %v", c.t.Name, h, c.hours)
		}
		if got := Score(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: score %v, want %v", c.t.Name, got, c.want)
		}
	}
}

// With 25 hours and integer ratings 1-5, a score
// is 25 x pain x undo: 450 can't happen (18 is not
// a product of two numbers from 1 to 5).
func TestReachableScores(t *testing.T) {
	seen := map[float64]bool{}
	for p := 1; p <= 5; p++ {
		for u := 1; u <= 5; u++ {
			tk := tagging
			tk.Pain, tk.Undo = p, u
			seen[math.Round(Score(tk))] = true
		}
	}
	if seen[450] {
		t.Fatal("450 should be unreachable at 25 h/wk")
	}
	if !seen[375] || !seen[625] {
		t.Fatal("375 and 625 should be reachable")
	}
}

func TestValidRejectsOutOfRange(t *testing.T) {
	for _, tk := range []Task{
		{Name: "pain 0", Pain: 0, Undo: 3},
		{Name: "pain 6", Pain: 6, Undo: 3},
		{Name: "undo 0", Pain: 3, Undo: 0},
		{Name: "undo 6", Pain: 3, Undo: 6},
		{Name: "unset"}, // zero values are not ratings
	} {
		if tk.Valid() == nil {
			t.Errorf("%s: want an error", tk.Name)
		}
	}
	for _, tk := range []Task{
		{Pain: 1, Undo: 1}, {Pain: 5, Undo: 5},
	} {
		if err := tk.Valid(); err != nil {
			t.Errorf("%+v: %v", tk, err)
		}
	}
}

func TestMultiplySinksIrreversible(t *testing.T) {
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

func TestDefaultsAreTheDocumentedOnes(t *testing.T) {
	edge := Task{PerWeek: 5, Minutes: 10, Pain: 3, Undo: 3}
	if !FirstProject(edge) {
		t.Error("undo 3 and five a week should pass")
	}
	edge.Undo = 2
	if FirstProject(edge) {
		t.Error("undo 2 should fail")
	}
	edge.Undo, edge.PerWeek = 3, 4
	if FirstProject(edge) {
		t.Error("four a week should fail")
	}
}

func TestRanking(t *testing.T) {
	ts := []Task{payments, checkCalls, tagging}
	slices.SortFunc(ts, func(x, y Task) int {
		return int(Score(y) - Score(x))
	})
	got := []string{ts[0].Name, ts[1].Name, ts[2].Name}
	want := []string{"ops@ tagging", "payments", "check calls"}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

// "With addition, huge volume can carry a task
// that's impossible to reverse." Multiplying stops it.
func TestAdditionLetsVolumeCarryIrreversible(t *testing.T) {
	big := Task{Name: "bulk payments", PerWeek: 400,
		Minutes: 6, Pain: 5, Undo: 1} // 40 h/wk
	add := func(t Task) float64 {
		return t.PerWeek*t.Minutes/60 +
			float64(t.Pain) + float64(t.Undo)
	}
	if add(big) <= add(tagging) {
		t.Fatalf("added: big %v should beat tagging %v",
			add(big), add(tagging))
	}
	if Score(big) >= Score(tagging) {
		t.Fatalf("multiplied: big %v should lose to tagging %v",
			Score(big), Score(tagging))
	}
}
