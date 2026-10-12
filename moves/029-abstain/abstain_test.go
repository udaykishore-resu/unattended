package abstain

import (
	"encoding/json"
	"testing"
)

// Every combination the reviewer listed. Only a real
// value with nothing listed, or null with the field
// listed, is accepted.
func TestConsistent(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{`{"po_number":"PO 4471","missing":[]}`, true},
		{`{"po_number":null,"missing":["po_number"]}`, true},
		// A value and a "missing" flag: contradiction.
		{`{"po_number":"PO 4471","missing":["po_number"]}`, false},
		// Null but not listed: did it look?
		{`{"po_number":null,"missing":[]}`, false},
		// Empty string is never a value: use null.
		{`{"po_number":"","missing":[]}`, false},
		{`{"po_number":"","missing":["po_number"]}`, false},
		{`{"po_number":"  ","missing":["po_number"]}`, false},
		// "N/A" passes the string type; Consistent refuses it.
		{`{"po_number":"N/A","missing":[]}`, false},
		{`{"po_number":"n/a","missing":["po_number"]}`, false},
		{`{"po_number":"none","missing":[]}`, false},
		// A few-shot example value leaked into output.
		{`{"po_number":"PO-EXAMPLE-1","missing":[]}`, false},
	}
	for _, c := range cases {
		var tn Tender
		if err := json.Unmarshal([]byte(c.in), &tn); err != nil {
			t.Fatal(err)
		}
		if err := Consistent(tn); (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.in, err)
		}
	}
}

// The schema step and Consistent both run in Accept.
// The schema step alone accepts a record that
// Consistent refuses, which is why both are needed.
func TestAcceptRunsBoth(t *testing.T) {
	for in, ok := range map[string]bool{
		`{"po_number":"PO 4471","missing":[]}`:            true,
		`{"po_number":null,"missing":["po_number"]}`:      true,
		`{"po_number":null,"missing":[]}`:                 false, // schema-valid, inconsistent
		`{"missing":["po_number"]}`:                       false, // key absent, not null
		`{"po_number":null}`:                              false,
		`{"po_number":42,"missing":[]}`:                   false,
		`{"po_number":null,"missing":["po_numbr"]}`:       false, // unknown field name
		`{"po_number":"PO 1","missing":["origin_zip"]}`:   false, // not in this schema's enum
		`{"po_number":"PO 1","missing":[],"rate":"2150"}`: false, // unknown key
	} {
		_, err := Accept([]byte(in))
		if (err == nil) != ok {
			t.Errorf("%s: err = %v", in, err)
		}
	}
}

func TestPerFieldPolicy(t *testing.T) {
	if a, any, err := Route([]string{"po_number"}); err != nil || !any || a != FollowUp {
		t.Fatalf("po: %v %v %v", a, any, err)
	}
	if a, _, err := Route([]string{"po_number", "origin_zip"}); err != nil || a != RejectWhole {
		t.Fatalf("origin: %v %v", a, err)
	}
	if _, any, err := Route(nil); err != nil || any {
		t.Fatalf("nothing missing: %v %v", any, err)
	}
	if a, _, err := Route([]string{"hazmat"}); err == nil || a != RejectWhole {
		t.Fatalf("unknown field must fail closed: %v %v", a, err)
	}
}

func TestSchemaAllowsNull(t *testing.T) {
	var s struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatal(err)
	}
	ty, ok := s.Properties["po_number"].Type.([]any)
	if !ok || len(ty) != 2 || ty[1] != "null" {
		t.Fatalf("po_number type = %v", s.Properties["po_number"].Type)
	}
}
