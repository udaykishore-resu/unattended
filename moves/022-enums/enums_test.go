package enums

import (
	"encoding/json"
	"testing"
)

type load struct {
	Equipment Equipment `json:"equipment"`
	Note      string    `json:"equipment_note"`
}

func TestEnum(t *testing.T) {
	cases := []struct {
		in   string
		want Equipment
		ok   bool
	}{
		{`" Reefer "`, "reefer", true}, // right answer, wrong shape
		{`"dry_van"`, "dry_van", true},
		{`"other"`, "other", true},
		{`"Refrigerated"`, "", false}, // a different string: repair it
		{`"REEFER (-10F)"`, "", false},
		{`42`, "", false},
	}
	for _, c := range cases {
		var l load
		err := json.Unmarshal([]byte(`{"equipment":`+c.in+`}`), &l)
		if (err == nil) != c.ok || l.Equipment != c.want {
			t.Errorf("%s: got %q, err %v", c.in, l.Equipment, err)
		}
	}
}

func TestOtherKeepsTheOddCaseVisible(t *testing.T) {
	var l load
	in := `{"equipment":"other","equipment_note":"conestoga"}`
	if err := json.Unmarshal([]byte(in), &l); err != nil {
		t.Fatal(err)
	}
	if l.Equipment != "other" || l.Note != "conestoga" {
		t.Fatalf("got %+v", l)
	}
}
