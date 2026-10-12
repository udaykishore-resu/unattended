// Package dedup flags duplicate loads only when an
// exact key and an embedding agree.
//
// Move 27: dedup with embeddings plus an exact key.
package dedup

import (
	"math"
	"slices"
	"strconv"
	"strings"
)

// Load holds normalized fields (Move 28) and an
// embedding of a short string built from them.
type Load struct {
	ShipperID, Pickup  string
	OriginZip, DestZip string
	Trucks             int
	Embedding          []float64
}

// Match is a three-value enum.
type Match int

const (
	New Match = iota
	AskHuman
	Duplicate
)

func ExactKey(l Load) (string, bool) {
	p := []string{l.ShipperID, l.Pickup,
		l.OriginZip, l.DestZip}
	if slices.Contains(p, "") || l.Trucks < 1 {
		return "", false // no key: blanks never match
	}
	p = append(p, strconv.Itoa(l.Trucks))
	return strings.Join(p, "|"), true
}

func Classify(a, b Load, near float64) Match {
	ka, okA := ExactKey(a)
	kb, okB := ExactKey(b)
	sameKey := okA && okB && ka == kb
	sim := cosine(a.Embedding, b.Embedding)
	switch {
	case sameKey && sim >= near:
		return Duplicate
	case sameKey || sim >= near:
		return AskHuman
	default:
		return New
	}
}

// cosine is "the usual dot-product helper".
func cosine(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range min(len(a), len(b)) {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
