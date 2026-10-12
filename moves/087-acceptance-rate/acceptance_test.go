package acceptance

import "testing"

const draft = "Hi Sam, load HF-1 is on schedule and will deliver Friday by noon. Thanks, Ines"

func TestFates(t *testing.T) {
	cases := []struct {
		sent string
		used bool
		want Fate
	}{
		{draft, false, Ignored},
		{"hi sam,  load HF-1 is on schedule and will deliver friday by noon. thanks, ines", true, Accepted},
		{"Hi Sam, load HF-1 is on schedule and will deliver Friday by 2pm. Thanks, Ines", true, Edited},
		{"Sam - the truck broke down in Joplin; new ETA Saturday. Sorry. Ines", true, Rewrote},
	}
	for _, c := range cases {
		if got := Classify(draft, c.sent, c.used); got != c.want {
			t.Errorf("%q: got %s, want %s (overlap %.2f)", c.sent, got, c.want,
				overlap(draft, c.sent))
		}
	}
}
