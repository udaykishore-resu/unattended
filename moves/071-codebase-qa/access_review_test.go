package codeqa

import "testing"

// The bot's index holds the finance repo; the asker
// is in dispatch only. The finance chunk never
// reaches the model, and a citation to it fails.
func TestAskerPermissionsNotBots(t *testing.T) {
	sha := "9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d"
	retrieved := []Chunk{
		{Path: "dispatch/score.go", Commit: sha, Start: 30, End: 60, Group: "eng"},
		{Path: "finance/payout.go", Commit: sha, Start: 1, End: 40, Group: "finance"},
	}
	shown := Visible(retrieved, map[string]bool{"eng": true})
	if len(shown) != 1 || shown[0].Group != "eng" {
		t.Fatalf("shown %+v", shown)
	}
	if err := Check("Payouts run in [finance/payout.go:12@9c1d2e3].", shown); err == nil {
		t.Fatal("citation to a hidden repo passed")
	}
	if err := Check("Scored in [dispatch/score.go:41@9c1d2e3].", shown); err != nil {
		t.Fatal(err)
	}
}

// Citations are checked against the exact indexed
// commit: a short SHA must be a prefix of it.
func TestCitationPinnedToIndexedCommit(t *testing.T) {
	got := []Chunk{{Path: "dispatch/score.go",
		Commit: "9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d", Start: 30, End: 60}}
	for ans, ok := range map[string]bool{
		"[dispatch/score.go:41@9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d]": true,
		"[dispatch/score.go:41@9c1d2e4]":                                  false, // next commit
		"[dispatch/score.go:41@1d2e3f4]":                                  false, // not a prefix
	} {
		if err := Check(ans, got); (err == nil) != ok {
			t.Errorf("%s: %v", ans, err)
		}
	}
}
