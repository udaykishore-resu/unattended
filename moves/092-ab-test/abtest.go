// Package abtest splits live traffic between two
// models by unit, with a kill switch.
//
// Move 92: A/B two models on live traffic safely.
package abtest

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/udaykishore-resu/unattended/llm"
)

type Split struct {
	A, B     Arm
	Percent  uint64 // share of units sent to B
	Salt     string // new salt per experiment
	Disabled func() bool
}

// For picks an arm by unit, not by call.
func (s Split) For(unit string) Arm {
	if s.Disabled != nil && s.Disabled() {
		return s.A
	}
	h := sha256.Sum256([]byte(s.Salt + ":" + unit))
	b := binary.BigEndian.Uint64(h[:8]) % 100
	if b < s.Percent {
		return s.B
	}
	return s.A
}

// Arm is a name plus an llm.Model; log the name on
// every call.
type Arm struct {
	Name  string
	Model llm.Model
}

// DiffCI is the 95% confidence interval for B's
// acceptance rate minus A's (normal approximation,
// fine at thousands of units per arm). If it spans
// zero, the test has not shown which arm is better.
// Run it per ticket type too, not only overall.
func DiffCI(okA, nA, okB, nB int) (lo, hi float64) {
	pa := float64(okA) / float64(nA)
	pb := float64(okB) / float64(nB)
	se := math.Sqrt(pa*(1-pa)/float64(nA) +
		pb*(1-pb)/float64(nB))
	d := pb - pa
	return d - 1.96*se, d + 1.96*se
}
