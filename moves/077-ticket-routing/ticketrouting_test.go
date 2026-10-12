package ticketrouting

import (
	"slices"
	"testing"
)

func TestUsable(t *testing.T) {
	shown := map[string]bool{"kb-12": true, "kb-40": true}
	plain := Ticket{ID: "T-1"}
	cases := []struct {
		name string
		t    Ticket
		r    Result
		ok   bool
	}{
		{"cites a shown article", plain, Result{Queue: "tracking", KBIDs: []string{"kb-12"}}, true},
		{"no source: a guess in a polite voice", plain, Result{Queue: "tracking"}, false},
		{"cites an article it never saw", plain, Result{Queue: "billing", KBIDs: []string{"kb-12", "kb-99"}}, false},
		{"unsure queue", plain, Result{Queue: "unsure", KBIDs: []string{"kb-12"}}, false},
		// Claims get no draft, whatever queue the model picked.
		{"claim from the form, model said billing", Ticket{ID: "T-2", Claim: true},
			Result{Queue: "billing", KBIDs: []string{"kb-12"}}, false},
		{"model routed to claims", plain, Result{Queue: "claims", KBIDs: []string{"kb-12"}}, false},
	}
	for _, c := range cases {
		if got := Usable(c.t, c.r, shown); got != c.ok {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

// Membership is not support: a draft that cites a
// shown article but says something the article does
// not still passes Usable. That is why a person sends.
func TestMembershipIsNotSupport(t *testing.T) {
	shown := map[string]bool{"kb-12": true} // kb-12: "Detention is billed after 2 hours."
	r := Result{Queue: "billing", KBIDs: []string{"kb-12"},
		Draft: "Detention is free for the first 6 hours."}
	if !Usable(Ticket{}, r, shown) {
		t.Fatal("expected pass: Usable checks ids, not content")
	}
}

func TestSendPathFlagsPromises(t *testing.T) {
	shown := map[string]bool{"kb-12": true}
	r := Result{Queue: "billing", KBIDs: []string{"kb-12"},
		Draft: "Sorry about that. We will Refund the fee and waive the next one."}
	draft, flags := Shown(Ticket{}, r, shown)
	if draft == "" || !slices.Equal(flags, []string{"refund", "waive"}) {
		t.Fatalf("draft=%q flags=%v", draft, flags)
	}
	// Claims tickets: no draft reaches the agent at all.
	draft, _ = Shown(Ticket{Claim: true}, r, shown)
	if draft != "" {
		t.Fatal("claims ticket got a draft")
	}
	// The limit of a word list, pinned so nobody oversells it.
	if got := Promises("We'll take care of that charge for you."); len(got) != 0 {
		t.Fatalf("word list now catches indirect promises: %v; update the book", got)
	}
}
