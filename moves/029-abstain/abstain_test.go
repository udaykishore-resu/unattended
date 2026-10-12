package abstain

import (
	"encoding/json"
	"testing"
)

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
		// Empty string counts as no value.
		{`{"po_number":"","missing":[]}`, false},
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
