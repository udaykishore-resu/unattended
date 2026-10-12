package dedup

import "testing"

func TestBothSignalsRequired(t *testing.T) {
	v := []float64{0.9, 0.1, 0.4}
	a := Load{"acme", "2026-03-12", "45402", "60432", 2, v}
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

// Reviewer cases: blanks, un-normalized fields and a
// different truck count must never produce Duplicate.
// Either signal alone still sends the pair to a person.
func TestKeyCannotCollapseDistinctLoads(t *testing.T) {
	v := []float64{0.9, 0.1, 0.4} // identical embeddings: sim = 1
	base := Load{"acme", "2026-03-12", "45402", "60432", 2, v}
	mut := func(f func(*Load)) Load { l := base; f(&l); return l }

	cases := map[string]Load{
		"no shipper":      mut(func(l *Load) { l.ShipperID = "" }),
		"no pickup date":  mut(func(l *Load) { l.Pickup = "" }),
		"no origin":       mut(func(l *Load) { l.OriginZip = "" }),
		"no truck count":  mut(func(l *Load) { l.Trucks = 0 }),
		"other quantity":  mut(func(l *Load) { l.Trucks = 3 }),
		"raw date":        mut(func(l *Load) { l.Pickup = "3/12/2026" }),
		"zip lost a zero": mut(func(l *Load) { l.DestZip = "6432" }),
	}
	for name, b := range cases {
		if got := Classify(base, b, 0.9); got != AskHuman {
			t.Errorf("%s: got %v, want AskHuman", name, got)
		}
	}
	// Two loads that are both blank in the same field
	// share no key, so they cannot match on paper.
	x := mut(func(l *Load) { l.ShipperID = "" })
	if _, ok := ExactKey(x); ok {
		t.Fatal("blank field produced a key")
	}
	if got := Classify(x, x, 0.9); got == Duplicate {
		t.Fatal("two blank-shipper loads became a duplicate")
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
