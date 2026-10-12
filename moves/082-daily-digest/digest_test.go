package digest

import (
	"testing"
	"time"
)

func TestPick(t *testing.T) {
	now := time.Date(2026, 10, 7, 6, 0, 0, 0, time.UTC)
	ago := func(d int) time.Time { return now.Add(-time.Duration(d)*24*time.Hour - time.Hour) }
	notes := []Note{
		{"retry-budgets.md", ago(90), ago(80)},                   // 90 days: back
		{"carrier-timeouts.md", ago(3), now.Add(-2 * time.Hour)}, // fresh
		{"week-old.md", ago(7), ago(7)},                          // 7 days: back
		{"random.md", ago(12), ago(12)},                          // not on a spacing day
	}
	fresh, old := Pick(notes, now)
	if len(fresh) != 1 || fresh[0].Path != "carrier-timeouts.md" {
		t.Fatalf("fresh = %+v", fresh)
	}
	if len(old) != 2 || old[0].Path != "retry-budgets.md" || old[1].Path != "week-old.md" {
		t.Fatalf("old = %+v", old)
	}
}

func TestKeepChecksBothPhrases(t *testing.T) {
	old := "July: retry budgets cap how often a client retries."
	recent := "Carrier timeouts spike when the EDI gateway retries every second."
	good := Link{Idea: "retries amplify load", OldPhrase: "retry  budgets", NewPhrase: "retries every second"}
	if !Keep(good, old, recent) {
		t.Fatal("real link dropped")
	}
	for name, l := range map[string]Link{
		"invented old phrase": {OldPhrase: "retry storms", NewPhrase: "retries every second"},
		"empty phrase":        {OldPhrase: "", NewPhrase: "retries every second"},
		"no link":             {NoLink: true},
	} {
		if Keep(l, old, recent) {
			t.Errorf("%s: kept", name)
		}
	}
}

func TestLineShowsCreatedDate(t *testing.T) {
	n := Note{Path: "retry-budgets.md", Created: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)}
	if got := Line(n, "retry budgets"); got != "retry-budgets.md (created 2026-07-09): retry budgets" {
		t.Fatal(got)
	}
}
