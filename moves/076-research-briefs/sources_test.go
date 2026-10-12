package briefs

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGatherFilters(t *testing.T) {
	now := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	long := strings.Repeat("Acme ships plastics by truckload. ", 20)
	srcs := []Source{
		{URL: "acme.test/about", Key: "acme.test", Title: "About Acme", Text: long, Fetched: now.Add(-time.Hour)},
		// Similar name, different company: the CFO mix-up.
		{URL: "acmeplastic.test/news", Key: "acmeplastic.test", Title: "New CFO", Text: long, Fetched: now},
		{URL: "acme.test/portal", Key: "acme.test", Title: "Sign in to Acme", Text: long, Fetched: now},
		{URL: "acme.test/old", Key: "acme.test", Title: "Team", Text: long, Fetched: now.AddDate(0, -3, 0)},
		{URL: "acme.test/blank", Key: "acme.test", Title: "Blog", Text: "Loading...", Fetched: now},
		{URL: "slow.test/profile", Key: "acme.test", Err: errors.New("timeout after 5s")},
	}
	keep, skipped := Gather(srcs, "acme.test", now, Limits{MinChars: 200, MaxAge: 30 * 24 * time.Hour})
	if len(keep) != 1 || keep[0].URL != "acme.test/about" {
		t.Fatalf("keep = %+v", keep)
	}
	for _, want := range []string{"different company key", "login page", "stale fetch", "too short", "fetch failed: timeout"} {
		if !slices.ContainsFunc(skipped, func(s string) bool { return strings.Contains(s, want) }) {
			t.Errorf("no skip reason %q in %q", want, skipped)
		}
	}
	h := Header("acme.test", keep, skipped)
	for _, want := range []string{"Company key: acme.test", "[S1] acme.test/about (fetched 2026-10-09)", "Skipped: slow.test/profile"} {
		if !strings.Contains(h, want) {
			t.Errorf("header missing %q:\n%s", want, h)
		}
	}
}

func TestDropUntagged(t *testing.T) {
	brief := "Acme makes plastic pallets [S1]. They are a great partner! " +
		"Their CFO joined in 2022 [S2]. Ask about Dallas"
	got := DropUntagged(brief)
	want := "Acme makes plastic pallets [S1]. Their CFO joined in 2022 [S2]."
	if got != want {
		t.Fatalf("got %q", got)
	}
	if DropUntagged("No tags at all. None.") != "" {
		t.Fatal("untagged text kept")
	}
}
