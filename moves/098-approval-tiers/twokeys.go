package tiers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Approval is one key. Who and Team come from the
// approver's login session, never from the request.
type Approval struct {
	Who, Team string
	Digest    string // the version they approved
}

func TwoKeys(p Proposal, as []Approval) error {
	if len(as) != 2 || as[0].Who == as[1].Who {
		return errors.New("need two different people")
	}
	fin := false
	for _, a := range as {
		if a.Digest != p.Digest() {
			return errors.New("approved another version")
		}
		if slices.Contains(p.Authors, a.Who) {
			return errors.New("an author cannot approve")
		}
		fin = fin || a.Team == "finance"
	}
	if !fin {
		return errors.New("need one finance key")
	}
	return nil
}

// Proposal is described in the book's prose: the exact
// arguments, the diff version the approver saw, and
// Authors, the proposer plus every editor.
type Proposal struct {
	Tool    string
	Input   json.RawMessage // exact arguments
	DiffVer int             // diff against current state
	Authors []string        // proposer and every editor
}

// Digest hashes the tool, exact arguments and diff
// version. Any edit after an approval changes it, so
// that approval no longer counts.
func (p Proposal) Digest() string {
	s := sha256.Sum256(fmt.Appendf(nil, "%s\n%d\n%s",
		p.Tool, p.DiffVer, p.Input))
	return hex.EncodeToString(s[:])
}
