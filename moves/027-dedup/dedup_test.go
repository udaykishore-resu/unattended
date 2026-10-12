package dedup

import "testing"

func TestBothSignalsRequired(t *testing.T) {
	v := []float64{0.9, 0.1, 0.4}
	a := Load{"acme", "2026-03-12", "45402", "60432", v}
	sameLoad := a
	// Same lane, same words, one week later: a different load.
	nextWeek := a
	nextWeek.Pickup = "2026-03-19"
	// Same key, but the text says something else.
	odd := a
	odd.Embedding = []float64{-0.1, 0.9, 0}

	cases := []struct {
		name string
		b    Load
		want Match
	}{
		{"same load", sameLoad, Duplicate},
		{"repeat lane next week", nextWeek, AskHuman},
		{"key only", odd, AskHuman},
		{"different", Load{ShipperID: "zeta", Embedding: []float64{0, 0, 1}}, New},
	}
	for _, c := range cases {
		if got := Classify(a, c.b, 0.9); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCosine(t *testing.T) {
	if c := cosine([]float64{1, 0}, []float64{1, 0}); c < 0.999 {
		t.Fatalf("identical = %v", c)
	}
	if c := cosine([]float64{1, 0}, []float64{0, 1}); c != 0 {
		t.Fatalf("orthogonal = %v", c)
	}
	if c := cosine(nil, []float64{1}); c != 0 {
		t.Fatalf("empty = %v", c)
	}
}
