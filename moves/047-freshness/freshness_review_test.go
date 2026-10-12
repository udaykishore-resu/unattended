package freshness

import (
	"testing"
	"time"
)

// Same boundary as the SQL in Move 44
// (expires_at >= now()): a chunk is live through its
// expiry instant and gone one tick later.
func TestExpiryBoundaryMatchesSQL(t *testing.T) {
	at := hit("edge", 3, 1, now)
	if got := Decay(now, []Hit{at}, 0); len(got) != 1 {
		t.Fatal("dropped at its expiry instant")
	}
	if got := Decay(now.Add(time.Nanosecond), []Hit{at}, 0); len(got) != 0 {
		t.Fatal("kept after expiry")
	}
}

// Decay is the second expiry check: an expired hit
// that slipped past search (a stale cache, or the
// rerank fallback returning raw candidates) is still
// dropped before the answering model.
func TestExpiredDroppedWhateverTheUpstream(t *testing.T) {
	stale := hit("fuel-31c", 3, 40, now.AddDate(0, 0, -14))
	for _, hl := range []time.Duration{0, 30 * 24 * time.Hour} {
		if got := Decay(now, []Hit{stale}, hl); len(got) != 0 {
			t.Fatalf("halfLife %v: expired hit returned", hl)
		}
	}
}
