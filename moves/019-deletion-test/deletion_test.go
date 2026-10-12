package deletion

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
)

// A fake eval: rule A is load-bearing, rule B is dead
// weight, rule C hurts.
func fakeEval(_ context.Context, sys string) (float64, error) {
	acc := 0.80
	if !strings.Contains(sys, "A: reply in French to Quebec") {
		acc -= 0.10
	}
	if strings.Contains(sys, "C: always say thanks twice") {
		acc -= 0.05
	}
	return acc, nil
}

func TestAblate(t *testing.T) {
	rules := []string{
		"A: reply in French to Quebec",
		"B: be helpful",
		"C: always say thanks twice",
	}
	var b strings.Builder
	if err := ablate(context.Background(), &b, "head", rules, fakeEval); err != nil {
		t.Fatal(err)
	}
	want := "baseline 0.750\n" +
		"-0.100 -A: reply in French to Quebec\n" +
		"+0.000 -B: be helpful\n" +
		"+0.050 -C: always say thanks twice\n"
	if b.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
	// slices.Clone matters: the caller's rules are untouched.
	if rules[0] != "A: reply in French to Quebec" || len(rules) != 3 {
		t.Fatalf("rules mutated: %q", rules)
	}
}

// An eval error stops the run and comes back to the
// caller, at the baseline or mid-ablation, so a
// half-finished report never passes as a full one.
func TestAblateReturnsEvalErrors(t *testing.T) {
	boom := errors.New("provider unavailable")
	rules := []string{"A: one", "B: two", "C: three"}
	for _, failOn := range []int{1, 3} { // baseline, second ablation
		calls := 0
		eval := func(context.Context, string) (float64, error) {
			calls++
			if calls == failOn {
				return 0, boom
			}
			return 0.8, nil
		}
		var b strings.Builder
		err := ablate(context.Background(), &b, "head", rules, eval)
		if !errors.Is(err, boom) {
			t.Fatalf("failOn %d: err = %v", failOn, err)
		}
		if calls != failOn {
			t.Fatalf("failOn %d: kept calling after the error (%d calls)", failOn, calls)
		}
	}
}

// Every ablation is compared with one baseline taken
// on the full prompt, and each variant drops exactly
// one rule and keeps the rest in order.
func TestEachVariantDropsOneRule(t *testing.T) {
	rules := []string{"A", "B", "C"}
	var seen []string
	eval := func(_ context.Context, sys string) (float64, error) {
		seen = append(seen, sys)
		return 1, nil
	}
	if err := ablate(context.Background(), io.Discard, "h", rules, eval); err != nil {
		t.Fatal(err)
	}
	want := []string{"h\nA\nB\nC", "h\nB\nC", "h\nA\nC", "h\nA\nB"}
	if !slices.Equal(seen, want) {
		t.Fatalf("prompts = %q", seen)
	}
}
