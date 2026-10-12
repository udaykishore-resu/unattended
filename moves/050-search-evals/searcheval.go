// Package searcheval scores retrieval against
// labeled real queries: recall at k and MRR.
//
// Move 50: build the eval set from real search
// logs.
package searcheval

import "slices"

// A Case holds a Query and Want, the doc IDs that
// answer it.
type Case struct {
	Query string
	Want  []string
}

// Score returns recall@k and MRR for a search
// function that returns ranked doc IDs.
func Score(k int, cases []Case,
	search func(string) []string) (rec, mrr float64) {
	for _, c := range cases {
		for i, id := range search(c.Query) {
			if !slices.Contains(c.Want, id) {
				continue
			}
			if i < k {
				rec++
			}
			mrr += 1 / float64(i+1)
			break
		}
	}
	n := float64(len(cases))
	return rec / n, mrr / n
}
