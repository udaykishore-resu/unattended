// Package searcheval scores retrieval against
// labeled real queries: recall at k and MRR.
//
// Move 50: build the eval set from real search
// logs.
package searcheval

import (
	"errors"
	"slices"
)

// A Case holds a Query and Want, the doc IDs that
// answer it.
type Case struct {
	Query string
	Want  []string
}

// Score returns recall@k and MRR. search returns one
// doc ID per result slot, best first.
func Score(k int, cases []Case,
	search func(string) []string,
) (rec, mrr float64, err error) {
	if len(cases) == 0 {
		return 0, 0, errors.New("empty eval set")
	}
	for _, c := range cases {
		// First slot holding any wanted doc.
		i := slices.IndexFunc(search(c.Query),
			func(id string) bool {
				return slices.Contains(c.Want, id)
			})
		if i < 0 {
			continue // a miss scores 0
		}
		if i < k {
			rec++ // once per query
		}
		mrr += 1 / float64(i+1)
	}
	n := float64(len(cases))
	return rec / n, mrr / n, nil
}
