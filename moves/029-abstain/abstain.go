// Package abstain makes "not present" a legal
// answer and checks that its two signals agree.
//
// Move 29: "not present" is an answer.
package abstain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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
	if t.PONumber != nil && placeholder(*t.PONumber) {
		return fmt.Errorf("po_number: %q: use null",
			*t.PONumber)
	}
	has := t.PONumber != nil
	if has == listed {
		return fmt.Errorf("po_number: value=%v "+
			"missing=%v", has, listed)
	}
	return nil
}

// placeholder catches values that mean "not present"
// but pass the string type: blanks, "N/A" and the
// fake values used in few-shot examples. Add new ones
// as reviewers find them.
func placeholder(s string) bool {
	v := strings.ToUpper(strings.TrimSpace(s))
	switch v {
	case "", "N/A", "NA", "NONE", "NULL", "UNKNOWN",
		"-", "--", "?":
		return true
	}
	return strings.HasPrefix(v, "PO-EXAMPLE")
}

// Accept is the whole gate for one record: the schema
// step (strict decode with both keys required; the
// schema alone cannot say "null only if listed"), then
// Consistent. Either failure gets Move 21's one repair.
func Accept(in []byte) (Tender, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return Tender{}, fmt.Errorf("decode: %w", err)
	}
	for _, k := range []string{"po_number", "missing"} {
		if _, ok := raw[k]; !ok {
			return Tender{}, fmt.Errorf("%s: required", k)
		}
	}
	var t Tender
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return Tender{}, fmt.Errorf("decode: %w", err)
	}
	allowed := missingEnum()
	for _, f := range t.Missing {
		if !slices.Contains(allowed, f) {
			return Tender{}, fmt.Errorf(
				"missing: unknown field %q", f)
		}
	}
	return t, Consistent(t)
}

// missingEnum reads the allowed "missing" values from
// the schema itself, so the two cannot drift.
func missingEnum() []string {
	var s struct {
		Properties struct {
			Missing struct {
				Items struct {
					Enum []string `json:"enum"`
				} `json:"items"`
			} `json:"missing"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		panic(err) // schema is a constant; tests cover it
	}
	return s.Properties.Missing.Items.Enum
}

// Action is what happens downstream when a field is
// legitimately missing. The policy is per field.
type Action int

const (
	FollowUp    Action = iota // keep the record, open a task
	RejectWhole               // record is unusable: review
)

// onMissing is Halcyon's per-field policy. A field not
// listed here may not be reported missing at all.
var onMissing = map[string]Action{
	"po_number":   FollowUp,    // ask before invoicing
	"origin_zip":  RejectWhole, // no origin, no load
	"pickup_date": RejectWhole,
}

// Route returns the strictest action over the fields
// the record lists as missing.
func Route(missing []string) (Action, bool, error) {
	act, any := FollowUp, false
	for _, f := range missing {
		a, ok := onMissing[f]
		if !ok {
			return RejectWhole, true, fmt.Errorf(
				"no policy for missing %q", f)
		}
		act, any = max(act, a), true
	}
	return act, any, nil
}
