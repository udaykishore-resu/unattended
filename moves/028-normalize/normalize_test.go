package normalize

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	chi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	want := time.Date(2026, 3, 4, 0, 0, 0, 0, chi)
	for _, raw := range []string{
		"2026-03-04", "03/04/2026", "3/4/2026", " Mar 4, 2026 ",
		"March 4, 2026", "4 Mar 2026",
	} {
		got, err := ParseDate(raw, chi)
		if err != nil || !got.Equal(want) {
			t.Errorf("%q: got %v, %v", raw, got, err)
		}
	}
}

// Parsed in the shipper's zone, a midnight pickup
// stays on its own day.
func TestZoneKeepsTheDay(t *testing.T) {
	chi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	got, _ := ParseDate("2026-03-04", chi)
	if got.In(chi).Day() != 4 {
		t.Fatalf("day = %d", got.In(chi).Day())
	}
}

func TestUnparsedIsAnError(t *testing.T) {
	for _, raw := range []string{"approx Thursday", "04.03.2026", ""} {
		if _, err := ParseDate(raw, time.UTC); err == nil {
			t.Errorf("%q parsed; want an error", raw)
		}
	}
}
