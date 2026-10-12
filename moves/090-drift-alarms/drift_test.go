package drift

import "testing"

var base = map[string]int{"detention": 300, "wismo": 250, "invoice": 250, "other": 100}

func psi(t *testing.T, b, n map[string]int) float64 {
	t.Helper()
	v, ok := PSI(b, n, 200)
	if !ok {
		t.Fatalf("unavailable for %v vs %v", b, n)
	}
	return v
}

func TestPSI(t *testing.T) {
	if got := psi(t, base, base); got != 0 {
		t.Fatalf("identical mix: %v", got)
	}
	// Normal Monday wobble.
	wobble := map[string]int{"detention": 290, "wismo": 260, "invoice": 245, "other": 105}
	if got := psi(t, base, wobble); got >= 0.1 {
		t.Fatalf("wobble PSI = %.3f, should not even say look", got)
	}
	// "other" climbs from a tenth to nearly a quarter
	// as detention emails land in it.
	drifted := map[string]int{"detention": 170, "wismo": 250, "invoice": 250, "other": 230}
	if got := psi(t, base, drifted); got < 0.1 {
		t.Fatalf("drifted PSI = %.3f, want at least 'look' (0.1)", got)
	}
	// A brand-new label shows as a large jump, not infinity.
	novel := map[string]int{"detention": 300, "wismo": 250, "invoice": 250, "other": 100, "hazmat": 90}
	if got := psi(t, base, novel); got < 0.25 || got > 10 {
		t.Fatalf("new label PSI = %.3f", got)
	}
	// A label that vanished (detention stops appearing)
	// is a large jump too.
	missing := map[string]int{"wismo": 400, "invoice": 350, "other": 150}
	if got := psi(t, base, missing); got < 0.25 || got > 10 {
		t.Fatalf("missing label PSI = %.3f", got)
	}
}

// An empty or tiny window is unavailable, not zero.
func TestTooFewIsUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		b, n   map[string]int
		minN   int
		wantOK bool
	}{
		{"empty now", base, nil, 200, false},
		{"empty base", nil, base, 200, false},
		{"empty, minN 0", base, map[string]int{}, 0, false},
		{"low volume", base, map[string]int{"other": 30}, 200, false},
		{"enough", base, base, 200, true},
	}
	for _, c := range cases {
		v, ok := PSI(c.b, c.n, c.minN)
		if ok != c.wantOK || (!ok && v != 0) {
			t.Errorf("%s: got %v, %v", c.name, v, ok)
		}
	}
}

// A reviewed history sets the alarm, not a generic
// number; weekly seasonality stays under it.
func TestThresholdFromHistory(t *testing.T) {
	weeks := []float64{0.02, 0.05, 0.11, 0.04}
	th := Threshold(weeks)
	if th < 0.132-1e-9 || th > 0.132+1e-9 {
		t.Fatalf("threshold = %v", th)
	}
	if Threshold(nil) != 0.1 {
		t.Fatal("no history: fall back to the 0.1 rule of thumb")
	}
}
