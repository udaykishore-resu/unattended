// Package drift compares an output label mix to a
// baseline with the population stability index.
//
// Move 90: drift alarms on output distributions.
package drift

import (
	"math"
	"slices"
)

// PSI compares two label distributions; base and
// now are counts per label. It returns false when
// either window has fewer than minN outputs.
func PSI(base, now map[string]int,
	minN int) (float64, bool) {
	nb, nn := total(base), total(now)
	if min(nb, nn) < float64(max(minN, 1)) {
		return 0, false // too few to tell
	}
	const eps = 1e-4 // avoid log(0)
	var psi float64
	for _, k := range union(base, now) {
		p := math.Max(float64(base[k])/nb, eps)
		q := math.Max(float64(now[k])/nn, eps)
		psi += (q - p) * math.Log(q/p)
	}
	return psi, true
}

// total sums the counts.
func total(m map[string]int) float64 {
	var n int
	for _, v := range m {
		n += v
	}
	return float64(n)
}

// union lists every label in either map, sorted so
// the float sum is the same on every run.
func union(a, b map[string]int) []string {
	var out []string
	for k := range a {
		out = append(out, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// Threshold is "a bit above the largest" PSI seen
// across reviewed normal weeks: 20% above it, and
// never below the 0.1 "look" rule of thumb.
func Threshold(normalWeeks []float64) float64 {
	return max(0.1, 1.2*slices.Max(append([]float64{0}, normalWeeks...)))
}
