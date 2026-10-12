package evidence

import (
	"encoding/json"
	"testing"
)

const email = `Hi team,
Please book pickup on
Thursday 3/12 at the Dayton dock.
Weight is 42,000 lbs.`

func TestGrounded(t *testing.T) {
	cases := map[string]bool{
		"pickup on thursday 3/12":        true, // wraps a line, other case
		"Weight is 42,000 lbs.":          true,
		"pickup on Friday 3/13":          false, // invented quote fails
		"":                               false, // no quote, no trust
		"   ":                            false,
		"the shipper confirmed by phone": false,
	}
	for ev, want := range cases {
		if got := grounded(email, ev); got != want {
			t.Errorf("grounded(%q) = %v, want %v", ev, got, want)
		}
	}
}

func TestFieldDecodes(t *testing.T) {
	var f Field[int]
	in := `{"value":42000,"confidence":"high","evidence":"Weight is 42,000 lbs."}`
	if err := json.Unmarshal([]byte(in), &f); err != nil {
		t.Fatal(err)
	}
	if f.Value != 42000 || !grounded(email, f.Evidence) {
		t.Fatalf("field = %+v", f)
	}
}
