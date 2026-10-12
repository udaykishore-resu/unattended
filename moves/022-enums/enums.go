// Package enums decodes a closed list of values and
// rejects anything else.
//
// Move 22: enums beat free text.
package enums

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Equipment string

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
