package enums

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
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

// The enum lives in three places; this test makes the
// Go map the one source. The schema is generated from
// it, and the SQL check must list exactly the same set.
func TestSQLCheckMatchesGo(t *testing.T) {
	b, err := os.ReadFile("equipment.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)IN \((.*?)\)`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no IN (...) list in equipment.sql")
	}
	var sqlVals []string
	for _, q := range regexp.MustCompile(`'([^']*)'`).FindAllSubmatch(m[1], -1) {
		sqlVals = append(sqlVals, string(q[1]))
	}
	slices.Sort(sqlVals)
	if !slices.Equal(sqlVals, SchemaEnum()) {
		t.Fatalf("SQL check %v != Go %v", sqlVals, SchemaEnum())
	}
}

func TestSchemaBuiltFromGo(t *testing.T) {
	var s struct {
		Type string   `json:"type"`
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(Schema(), &s); err != nil {
		t.Fatal(err)
	}
	if s.Type != "string" || !slices.Equal(s.Enum, SchemaEnum()) ||
		len(s.Enum) != len(validEquip) {
		t.Fatalf("schema = %s", Schema())
	}
}

// Reviewer cases: null, "", unknown, non-string and a
// missing key all produce a problem for the one repair;
// none is silently defaulted to a value.
func TestInvalidGoesToRepairNotDefault(t *testing.T) {
	for _, in := range []string{
		`{"equipment":null}`,
		`{"equipment":""}`,
		`{"equipment":"   "}`,
		`{"equipment":"Refrigerated"}`,
		`{"equipment":42}`,
		`{"equipment":true}`,
		`{"equipment":["reefer"]}`,
		`{"equipment":{"type":"reefer"}}`,
		`{}`,
		`{"equipment":"other"}`, // other needs a note
		`{"equipment":"other","equipment_note":"  "}`,
		`{"equipment":"reefer","temp":"-10F"}`, // unknown field
	} {
		l, probs := Parse([]byte(in))
		if len(probs) == 0 {
			t.Errorf("%s: accepted as %+v", in, l)
		}
	}
	l, probs := Parse([]byte(`{"equipment":" Reefer "}`))
	if len(probs) != 0 || l.Equipment != "reefer" {
		t.Fatalf("good input: %+v %v", l, probs)
	}
	l, probs = Parse([]byte(`{"equipment":"other","equipment_note":"conestoga"}`))
	if len(probs) != 0 || l.Equipment != "other" {
		t.Fatalf("other with note: %+v %v", l, probs)
	}
}

func TestOtherShareAlert(t *testing.T) {
	week := make([]Equipment, 100)
	for i := range week {
		week[i] = "reefer"
	}
	if _, alert := OtherShare(week, 0.05); alert {
		t.Fatal("no other, no alert")
	}
	for i := range 6 {
		week[i] = "other"
	}
	if s, alert := OtherShare(week, 0.05); !alert || s != 0.06 {
		t.Fatalf("share=%v alert=%v", s, alert)
	}
	if _, alert := OtherShare(nil, 0.05); alert {
		t.Fatal("empty week must not alert")
	}
}
