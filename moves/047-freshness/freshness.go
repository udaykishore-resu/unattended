// Package freshness drops expired hits and ages the
// rest, after the rerank.
//
// Move 47: freshness metadata and expiry dates.
package freshness

import (
	"math"
	"sort"
	"time"
)

// Chunk is the chunk from Move 43, trimmed.
type Chunk struct {
	ID, Text string
	Updated  time.Time // reviewed date
}

// A Hit embeds the Chunk from Move 43 and adds
// Expires, Owner, Raw (the rerank score) and the
// decayed Score.
type Hit struct {
	Chunk
	Expires time.Time
	Owner   string
	Raw     float64
	Score   float64
}

// Decay runs after the rerank. It drops expired
// hits and ages Score; Raw stays as scored.
func Decay(now time.Time, hits []Hit,
	halfLife time.Duration) []Hit {
	var out []Hit
	for _, h := range hits {
		if !h.Expires.IsZero() && now.After(h.Expires) {
			continue
		}
		h.Score = h.Raw
		if halfLife > 0 {
			age := now.Sub(h.Updated).Hours()
			h.Score *= math.Exp2(-age / halfLife.Hours())
		}
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}
