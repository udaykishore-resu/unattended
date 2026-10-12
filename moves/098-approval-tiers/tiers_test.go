package tiers

import "testing"

func TestTierFor(t *testing.T) {
	cases := []struct {
		name string
		a    Action
		want Tier
	}{
		{"internal note", Action{Tool: "add_note", Reversible: true}, Auto},
		{"email to dispatch channel", Action{Tool: "send_email", Reversible: false}, Review},
		{"same tool to a shipper", Action{Tool: "send_email", Outside: 1}, Review},
		{"refund", Action{Tool: "refund", Cents: 190_000, Reversible: true}, TwoKey},
		{"$4,999 refund still moves money", Action{Tool: "refund", Cents: 499_900}, TwoKey},
		{"over $5,000", Action{Tool: "refund", Cents: 570_000}, Never},
	}
	for _, c := range cases {
		if got := TierFor(c.a); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Reversible defaults to false: you cannot unsend.
func TestDefaultIsNotReversible(t *testing.T) {
	if TierFor(Action{Tool: "send_email"}) != Review {
		t.Fatal("a zero-value action must not be Auto")
	}
}
