// Package sampling picks a deterministic, stratified
// slice of outputs for human review, forever.
//
// Move 91: sample a slice for human review, forever.
package sampling

import (
	"crypto/sha256"
	"encoding/binary"
)

var Rates = map[string]float64{
	"damage_claim": 0.20, "detention": 0.05,
	"rate_request": 0.03,
	"other":        0.15, // catch-all: misses hide here
}

// Pick is deterministic: same id, same answer.
func Pick(id, stratum, salt string) bool {
	r, ok := Rates[stratum]
	if !ok {
		r = 0.10 // unknown label: look hard
	}
	r = max(r, 0.01) // floor for every stratum
	h := sha256.Sum256([]byte(salt + ":" + id))
	u := binary.BigEndian.Uint64(h[:8])
	return float64(u)/float64(^uint64(0)) < r
}
