// Package abtest splits live traffic between two
// models by unit, with a kill switch.
//
// Move 92: A/B two models on live traffic safely.
package abtest

import (
	"crypto/sha256"
	"encoding/binary"

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
	h := sha256.Sum256([]byte(s.Salt + unit))
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
