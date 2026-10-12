// Package enums decodes a closed list of values and
// rejects anything else.
//
// Move 22: enums beat free text.
package enums

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Equipment string

// validEquip is the single source of truth. The JSON
// Schema enum is built from it (SchemaEnum), and a
// test fails if the Postgres CHECK in equipment.sql
// lists anything else.
var validEquip = map[Equipment]bool{
	"dry_van": true, "reefer": true, "flatbed": true,
	"step_deck": true, "other": true,
}

func (e *Equipment) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v := Equipment(strings.ToLower(strings.TrimSpace(s)))
	if !validEquip[v] {
		return fmt.Errorf("equipment %q not allowed", s)
	}
	*e = v
	return nil
}

// SchemaEnum returns the allowed values, sorted, for
// the "enum" keyword of the JSON Schema.
func SchemaEnum() []string {
	var out []string
	for v := range validEquip {
		out = append(out, string(v))
	}
	slices.Sort(out)
	return out
}

// Schema is the equipment property of the tender
// schema, generated from validEquip.
func Schema() json.RawMessage {
	b, err := json.Marshal(map[string]any{
		"type": "string", "enum": SchemaEnum(),
	})
	if err != nil {
		panic(err) // a map of strings always marshals
	}
	return b
}

// Load is the slice of the tender this move cares
// about.
type Load struct {
	Equipment Equipment `json:"equipment"`
	Note      string    `json:"equipment_note"`
}

// Parse decodes strictly and runs the rules, one
// problem string per failure, in the shape Move 21's
// repair step expects. Nothing invalid is defaulted:
// a missing key never reaches UnmarshalJSON and
// leaves "", which the rules reject.
func Parse(in []byte) (Load, []string) {
	var l Load
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&l); err != nil {
		return Load{}, []string{"decode: " + err.Error()}
	}
	var probs []string
	if l.Equipment == "" {
		probs = append(probs, "equipment: required")
	}
	// Policy: other is legal only with a note that
	// describes the trailer.
	if l.Equipment == "other" &&
		strings.TrimSpace(l.Note) == "" {
		probs = append(probs,
			"equipment_note: required when equipment is other")
	}
	return l, probs
}

// OtherShare reports whether the share of "other" in
// one week's values is above the alert threshold.
// Halcyon starts at 5% (a default, not a law).
func OtherShare(week []Equipment, limit float64) (
	float64, bool) {
	if len(week) == 0 {
		return 0, false
	}
	n := 0
	for _, e := range week {
		if e == "other" {
			n++
		}
	}
	share := float64(n) / float64(len(week))
	return share, share > limit
}
