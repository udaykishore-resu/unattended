package cite

import "testing"

var sent = map[string]string{
	"brightline-7": "Shipper may cancel without charge if pickup " +
		"is more than four (4) hours late.",
	"acme-2": "Acme may cancel without charge if pickup is " +
		"more than two hours late.",
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		a    Answer
		ok   bool
	}{
		{"good quote, wrapped and cased", Answer{Citations: []Citation{{"brightline-7",
			"MORE THAN FOUR (4)\n hours late"}}}, true},
		{"abstain is legal", Answer{Abstain: true}, true},
		{"no citation", Answer{Text: "yes, two hours"}, false},
		{"chunk not sent", Answer{Citations: []Citation{{"brightline-9",
			"more than two hours late"}}}, false},
		// The two-hour rule belongs to another customer.
		{"quote from the wrong chunk", Answer{Citations: []Citation{{"brightline-7",
			"more than two hours late"}}}, false},
		{"too short to prove anything", Answer{Citations: []Citation{{"brightline-7",
			"late"}}}, false},
	}
	for _, c := range cases {
		if err := Check(c.a, sent); (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
}

func TestAbstainInCode(t *testing.T) {
	if !AbstainInCode(0, 0) || !AbstainInCode(1, 3) || AbstainInCode(3, 2) {
		t.Fatal("code exits wrong")
	}
}
