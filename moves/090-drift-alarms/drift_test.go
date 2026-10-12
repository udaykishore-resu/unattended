package drift

import "testing"

func TestPSI(t *testing.T) {
	base := map[string]int{"detention": 300, "wismo": 250, "invoice": 250, "other": 100}
	if got := PSI(base, base); got != 0 {
		t.Fatalf("identical mix: %v", got)
	}
	// Normal Monday wobble.
	wobble := map[string]int{"detention": 290, "wismo": 260, "invoice": 245, "other": 105}
	if got := PSI(base, wobble); got >= 0.1 {
		t.Fatalf("wobble PSI = %.3f, should not even say look", got)
	}
	// "other" climbs from a tenth to nearly a quarter
	// as detention emails land in it.
	drifted := map[string]int{"detention": 170, "wismo": 250, "invoice": 250, "other": 230}
	if got := PSI(base, drifted); got < 0.1 {
		t.Fatalf("drifted PSI = %.3f, want at least 'look' (0.1)", got)
	}
	// A brand-new label shows as a large jump, not infinity.
	novel := map[string]int{"detention": 300, "wismo": 250, "invoice": 250, "other": 100, "hazmat": 90}
	if got := PSI(base, novel); got < 0.25 || got > 10 {
		t.Fatalf("new label PSI = %.3f", got)
	}
	if PSI(nil, base) != 0 {
		t.Fatal("empty baseline should be 0")
	}
}
