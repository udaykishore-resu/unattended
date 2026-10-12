package oneverb

import (
	"encoding/json"
	"testing"
)

// The CI test from the book, run over every tool
// this agent registers.
func TestToolNamesAreVerbNoun(t *testing.T) {
	for _, tool := range Tools {
		if err := CheckName(tool.Name); err != nil {
			t.Error(err)
		}
	}
}

// How the first-underscore split treats awkward
// names, including compound verbs.
func TestCheckNameCases(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"set_pickup_window", true}, // noun may hold underscores
		{"find_load_by_ref", true},
		{"cancel_load", true},              // a fine name; this agent just doesn't hold it
		{"manage_load", false},             // mode switch, not a verb
		{"update_delivery_windows", false}, // not on the list
		{"check_in_driver", false},         // compound verb splits as "check"
		{"getload", false},                 // no underscore
		{"get_", false},                    // no noun
		{"_load", false},                   // no verb
		{"Get_load", false},                // case matters
	}
	for _, c := range cases {
		if err := CheckName(c.name); (err == nil) != c.ok {
			t.Errorf("CheckName(%q) = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

// The schema shapes the call: no status, rate or
// carrier field, and unknown keys are not allowed.
// It is not the permission; see Move 33's executor.
func TestNarrowSchema(t *testing.T) {
	var s struct {
		AdditionalProperties *bool          `json:"additionalProperties"`
		Properties           map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(setPickupWindow.InputSchema, &s); err != nil {
		t.Fatal(err)
	}
	if s.AdditionalProperties == nil || *s.AdditionalProperties {
		t.Fatal("additionalProperties must be false")
	}
	for _, k := range []string{"status", "rate", "carrier", "action"} {
		if _, ok := s.Properties[k]; ok {
			t.Errorf("set_pickup_window must not take %q", k)
		}
	}
	for _, tool := range Tools {
		if tool.Name == "cancel_load" {
			t.Fatal("this agent must not hold cancel_load")
		}
	}
}
