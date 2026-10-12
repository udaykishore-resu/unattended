package ticketrouting

import "testing"

func TestUsable(t *testing.T) {
	shown := map[string]bool{"kb-12": true, "kb-40": true}
	cases := []struct {
		name string
		r    Result
		ok   bool
	}{
		{"cites a shown article", Result{Queue: "tracking", KBIDs: []string{"kb-12"}}, true},
		{"no source: a guess in a polite voice", Result{Queue: "tracking"}, false},
		{"cites an article it never saw", Result{Queue: "billing", KBIDs: []string{"kb-12", "kb-99"}}, false},
		{"unsure queue", Result{Queue: "unsure", KBIDs: []string{"kb-12"}}, false},
	}
	for _, c := range cases {
		if got := Usable(c.r, shown); got != c.ok {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}
