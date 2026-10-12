package freshness

import (
	"math"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

func hit(id string, raw float64, ageDays int, expires time.Time) Hit {
	return Hit{Chunk: Chunk{ID: id, Updated: now.AddDate(0, 0, -ageDays)},
		Raw: raw, Expires: expires}
}

// Marcus moved fuel from 31 to 34 cents. The old
// table expired; only the new one can win.
func TestExpiredNeverReturned(t *testing.T) {
	old := hit("fuel-31c", 3, 40, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	cur := hit("fuel-34c", 3, 14, time.Time{})
	got := Decay(now, []Hit{old, cur}, 30*24*time.Hour)
	if len(got) != 1 || got[0].ID != "fuel-34c" {
		t.Fatalf("got %+v", got)
	}
}

func TestHalfLife(t *testing.T) {
	h := hit("rates", 2, 30, time.Time{})
	got := Decay(now, []Hit{h}, 30*24*time.Hour)
	if math.Abs(got[0].Score-1) > 1e-9 || got[0].Raw != 2 {
		t.Fatalf("score %v raw %v; want halved score, raw kept", got[0].Score, got[0].Raw)
	}
}

// Contracts get no decay, just a hard expiry.
// Without the zero guard every contract would sink.
func TestZeroHalfLifeMeansNoDecay(t *testing.T) {
	h := hit("contract", 3, 700, time.Time{})
	got := Decay(now, []Hit{h}, 0)
	if got[0].Score != 3 {
		t.Fatalf("score = %v", got[0].Score)
	}
}

func TestNewerWinsAtEqualRaw(t *testing.T) {
	got := Decay(now, []Hit{hit("old", 3, 90, time.Time{}), hit("new", 3, 1, time.Time{})},
		30*24*time.Hour)
	if got[0].ID != "new" {
		t.Fatalf("order = %s, %s", got[0].ID, got[1].ID)
	}
}
