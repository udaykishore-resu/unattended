package tiers

import (
	"encoding/json"
	"testing"
)

func refund() Proposal {
	return Proposal{Tool: "refund",
		Input:   json.RawMessage(`{"invoice":"40117","cents":190000}`),
		DiffVer: 3, Authors: []string{"agent:billing-1.4.2", "ines"}}
}

func key(who, team string, p Proposal) Approval {
	return Approval{Who: who, Team: team, Digest: p.Digest()}
}

func TestTwoKeysIndependence(t *testing.T) {
	p := refund()
	edited := refund()
	edited.Input = json.RawMessage(`{"invoice":"40117","cents":980000}`)
	newer := refund()
	newer.DiffVer = 4

	cases := []struct {
		name string
		as   []Approval
		ok   bool
	}{
		{"two people, one finance", []Approval{key("marcus", "finance", p), key("dana", "ops", p)}, true},
		{"same person twice", []Approval{key("marcus", "finance", p), key("marcus", "finance", p)}, false},
		{"one key only", []Approval{key("marcus", "finance", p)}, false},
		{"three keys", []Approval{key("marcus", "finance", p), key("dana", "ops", p), key("theo", "platform", p)}, false},
		{"editor approves", []Approval{key("marcus", "finance", p), key("ines", "support", p)}, false},
		{"proposer approves", []Approval{key("marcus", "finance", p), key("agent:billing-1.4.2", "ops", p)}, false},
		{"no finance key", []Approval{key("dana", "ops", p), key("theo", "platform", p)}, false},
		{"approved before an edit", []Approval{key("marcus", "finance", edited), key("dana", "ops", p)}, false},
		{"approved an older diff", []Approval{key("marcus", "finance", p), key("dana", "ops", newer)}, false},
	}
	for _, c := range cases {
		if err := TwoKeys(p, c.as); (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
}

// An edit after both keys turn invalidates them.
func TestEditAfterApprovalNeedsNewKeys(t *testing.T) {
	p := refund()
	as := []Approval{key("marcus", "finance", p), key("dana", "ops", p)}
	if err := TwoKeys(p, as); err != nil {
		t.Fatal(err)
	}
	p.Input = json.RawMessage(`{"invoice":"40117","cents":190001}`)
	p.Authors = append(p.Authors, "dana")
	if TwoKeys(p, as) == nil {
		t.Fatal("old keys still open the edited action")
	}
}
