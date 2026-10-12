package allowlist

import (
	"encoding/json"
	"strings"
	"testing"
)

// One passing and one failing case per entry, plus
// the tool nobody thought to deny.
func TestGate(t *testing.T) {
	cases := []struct {
		tool, in string
		ok       bool
	}{
		{"get_shipment", `{"id":"HF-1"}`, true},
		{"add_note", `{"text":"carrier confirmed"}`, true},
		{"add_note", `{"text":"` + strings.Repeat("x", 2001) + `"}`, false},
		{"set_status", `{"status":"delivered"}`, true},
		{"set_status", `{"status":"cancelled"}`, false},
		{"set_status", `{"status":7}`, false},
		{"set_status", `not json`, false},
		{"cancel_load", `{"id":"HF-1"}`, false}, // new MCP tool: denied by default
		{"delete_shipment", `{}`, false},
	}
	for _, c := range cases {
		err := Gate(c.tool, json.RawMessage(c.in))
		if (err == nil) != c.ok {
			t.Errorf("%s %s: err = %v", c.tool, c.in, err)
		}
	}
}

// The test the book asks for: no wildcard, and every
// entry has a test case above.
func TestNoWildcards(t *testing.T) {
	tested := map[string]bool{"get_shipment": true, "add_note": true, "set_status": true}
	for name := range Allowed {
		if strings.ContainsAny(name, "*?") {
			t.Errorf("wildcard entry %q", name)
		}
		if !tested[name] {
			t.Errorf("entry %q has no test case", name)
		}
	}
}
