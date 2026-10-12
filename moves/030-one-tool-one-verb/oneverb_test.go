package oneverb

import (
	"encoding/json"
	"strings"
	"testing"
)

// The CI test from the book: split each name at the
// first underscore and check the verb.
func TestToolNamesAreVerbNoun(t *testing.T) {
	for _, tool := range Tools {
		verb, noun, ok := strings.Cut(tool.Name, "_")
		if !ok || noun == "" || !Verbs[verb] {
			t.Errorf("tool %q is not verb_noun from the list", tool.Name)
		}
	}
	if Verbs[strings.SplitN("manage_load", "_", 2)[0]] {
		t.Error("manage is a mode switch, not a verb")
	}
}

// The schema is the permission: no status, rate or
// carrier field, and unknown keys are rejected.
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
