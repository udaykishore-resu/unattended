// Package hybrid merges keyword and vector result
// lists by rank.
//
// Move 44: hybrid search: keyword plus vectors.
package hybrid

import (
	"maps"
	"slices"
	"sort"
)

// Fuse merges ranked ID lists with reciprocal
// rank fusion. k=60 is the common default.
func Fuse(k float64, lists ...[]string) []string {
	score := map[string]float64{}
	for _, list := range lists {
		for rank, id := range list {
			score[id] += 1 / (k + float64(rank+1))
		}
	}
	ids := slices.Collect(maps.Keys(score))
	sort.Slice(ids, func(i, j int) bool {
		a, b := score[ids[i]], score[ids[j]]
		if a != b {
			return a > b
		}
		return ids[i] < ids[j]
	})
	return ids
}
