package deletion

import (
	"context"
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
