package allowlist

import (
	"encoding/json"
	"strings"
	"testing"
)

// One passing and one failing case per entry, plus
// the tool nobody thought to deny.
var cases = []struct {
	tool, in string
	ok       bool
}{
	{"get_shipment", `{"id":"HF-1"}`, true},
	{"get_shipment", `not json`, true}, // nil check: args unchecked; the tool's own decode and authz refuse it
	{"add_note", `{"text":"carrier confirmed"}`, true},
	{"add_note", `{"text":"` + strings.Repeat("x", 2001) + `"}`, false},
	{"set_status", `{"status":"delivered"}`, true},
	{"set_status", `{"status":"cancelled"}`, false},
	{"set_status", `{"status":7}`, false},
	{"set_status", `not json`, false},
	{"cancel_load", `{"id":"HF-1"}`, false}, // new MCP tool: denied by default
	{"delete_shipment", `{}`, false},
}

func TestGate(t *testing.T) {
	for _, c := range cases {
		err := Gate(c.tool, json.RawMessage(c.in))
		if (err == nil) != c.ok {
			t.Errorf("%s %s: err = %v", c.tool, c.in, err)
		}
	}
}

// The test the book asks for: no wildcard, and every
// entry has one passing and one failing case above
// (a nil check has no failing argument case).
func TestNoWildcards(t *testing.T) {
	pass, fail := map[string]bool{}, map[string]bool{}
	for _, c := range cases {
		if c.ok {
			pass[c.tool] = true
		} else {
			fail[c.tool] = true
		}
	}
	for name, chk := range Allowed {
		if strings.ContainsAny(name, "*?") {
			t.Errorf("wildcard entry %q", name)
		}
		if !pass[name] || (chk != nil && !fail[name]) {
			t.Errorf("entry %q lacks a passing or failing case", name)
		}
	}
}
