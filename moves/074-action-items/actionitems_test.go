package actionitems

import "testing"

const notes = "Dana: dock schedule is a mess.\nTheo: I’ll  fix the\ndock schedule by Friday."

var roster = map[string]bool{"Theo": true, "Dana": true, "Ines": true}

func TestCheck(t *testing.T) {
	cases := []struct {
		it   Item
		want string
	}{
		// Line break and curly quote still match.
		{Item{Task: "Fix dock schedule", Owner: "Theo", Due: "by Friday",
			Quote: "I'll fix the dock schedule by Friday"}, ""},
		// Someone who left in March.
		{Item{Owner: "Priya", Quote: "I'll fix the dock schedule"}, "owner not on roster"},
		// Paraphrased: not what was said.
		{Item{Owner: "Dana", Quote: "Dana will own the dock schedule"}, "quote not found in notes"},
	}
	for _, c := range cases {
		if got := Check(c.it, roster, notes); got != c.want {
			t.Errorf("%+v: got %q, want %q", c.it, got, c.want)
		}
	}
}

func TestKeyIgnoresRewordedTask(t *testing.T) {
	a := Item{Task: "Fix the dock schedule", Owner: "Theo", Quote: "I'll fix the dock schedule"}
	b := Item{Task: "Repair dock scheduling", Owner: "Theo", Quote: "I’ll  fix the dock schedule"}
	if Key("m-42", a) != Key("m-42", b) {
		t.Fatal("rerun would file a duplicate ticket")
	}
}
