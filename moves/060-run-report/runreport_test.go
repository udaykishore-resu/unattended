package runreport

import (
	"strings"
	"testing"
	"time"
)

// The stale-load sweep exited zero six nights in a
// row with "Seen 0". The comparison catches night one.
func TestSeenZeroIsTheLoudestLine(t *testing.T) {
	recent := []Report{{Seen: 402}, {Seen: 397}, {Seen: 411}, {Seen: 389}}
	got := Alarms(Report{Job: "stale-load-sweep", Seen: 0}, recent, 5)
	if len(got) != 1 || !strings.Contains(got[0], "seen 0, rolling median 402") {
		t.Fatalf("alarms = %v", got)
	}
	if a := Alarms(Report{Seen: 395, Acted: 395}, recent, 5); len(a) != 0 {
		t.Fatalf("normal night alarmed: %v", a)
	}
}

func TestTooManyFailures(t *testing.T) {
	r := Report{Seen: 400, Acted: 397, Failed: []string{"a", "b", "c"}}
	if got := Alarms(r, nil, 2); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

// Every item seen lands in one bucket; retried items
// that finished count where they finished, and items
// left for later are Pending, so totals reconcile.
func TestBucketsReconcile(t *testing.T) {
	r := Report{
		Seen:    400,
		Acted:   352,
		Skipped: map[string]int{"no carrier email on file": 31, "already chased": 10},
		Failed:  []string{"L-1", "L-2"},
		Pending: []string{"L-3", "L-4", "L-5", "L-6", "L-7"}, // DLQ / review
		Retries: 17,                                          // attempts, not items
	}
	if d := r.Unbalanced(); d != 0 {
		t.Fatalf("unbalanced by %d", d)
	}
	if a := Alarms(r, nil, 5); len(a) != 0 {
		t.Fatalf("alarms %v", a)
	}
	// Forget to count the pending items and the report says so.
	r.Pending = nil
	got := Alarms(r, nil, 5)
	if len(got) != 1 || !strings.Contains(got[0], "unbalanced: 5") {
		t.Fatalf("alarms = %v", got)
	}
}

func TestRuntimeDoubled(t *testing.T) {
	t0 := time.Now()
	recent := []Report{
		{Seen: 400, Start: t0, End: t0.Add(3 * time.Second)},
		{Seen: 400, Start: t0, End: t0.Add(4 * time.Second)},
		{Seen: 400, Start: t0, End: t0.Add(3 * time.Second)},
	}
	slow := Report{Seen: 400, Acted: 400, Start: t0, End: t0.Add(40 * time.Second)}
	if got := Alarms(slow, recent, 5); len(got) != 1 || !strings.Contains(got[0], "runtime") {
		t.Fatalf("alarms = %v", got)
	}
}
