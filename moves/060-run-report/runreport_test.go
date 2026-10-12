package runreport

import (
	"strings"
	"testing"
)

// The stale-load sweep exited zero six nights in a
// row with "Seen 0". The comparison catches night one.
func TestSeenZeroIsTheLoudestLine(t *testing.T) {
	recent := []Report{{Seen: 402}, {Seen: 397}, {Seen: 411}, {Seen: 389}}
	got := Alarms(Report{Job: "stale-load-sweep", Seen: 0}, recent, 5)
	if len(got) != 1 || !strings.Contains(got[0], "seen 0, rolling median 402") {
		t.Fatalf("alarms = %v", got)
	}
	if a := Alarms(Report{Seen: 395}, recent, 5); len(a) != 0 {
		t.Fatalf("normal night alarmed: %v", a)
	}
}

func TestTooManyFailures(t *testing.T) {
	r := Report{Seen: 400, Failed: []string{"a", "b", "c"}}
	if got := Alarms(r, nil, 2); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}
