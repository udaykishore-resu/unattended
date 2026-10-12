// Package abstain makes "not present" a legal
// answer and checks that its two signals agree.
//
// Move 29: "not present" is an answer.
package abstain

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Tender has PONumber *string and Missing []string.
type Tender struct {
	PONumber *string  `json:"po_number"`
	Missing  []string `json:"missing"`
}

var schema = json.RawMessage(`{
  "type": "object",
  "required": ["po_number", "missing"],
  "properties": {
    "po_number": {"type": ["string", "null"]},
    "missing": {"type": "array",
      "items": {"enum": ["po_number"]}}
  }
}`)

func Consistent(t Tender) error {
	listed := slices.Contains(t.Missing, "po_number")
	has := t.PONumber != nil && *t.PONumber != ""
	if has == listed {
		return fmt.Errorf("po_number: value=%v "+
			"missing=%v", has, listed)
	}
	return nil
}
