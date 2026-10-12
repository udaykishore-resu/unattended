package timeline

import (
	"testing"
	"time"
)

func TestDuplicateAndUnknownIDs(t *testing.T) {
	t0 := time.Date(2026, 3, 9, 14, 12, 0, 0, time.UTC)
	if _, err := Index([]Event{{"chat:1712", t0}, {"chat:1712", t0.Add(time.Minute)}}); err == nil {
		t.Fatal("duplicate event id accepted")
	}
	at, err := Index([]Event{{"chat:1712", t0}})
	if err != nil {
		t.Fatal(err)
	}
	// The same ID cited twice in one entry is harmless.
	ts, err := Stamp([]Entry{{EventIDs: []string{"chat:1712", "chat:1712"}}}, at)
	if err != nil || !ts[0].Equal(t0) {
		t.Fatal(ts, err)
	}
	// One unknown ID among good ones fails the draft.
	if _, err := Stamp([]Entry{{EventIDs: []string{"chat:1712", "chat:9999"}}}, at); err == nil {
		t.Fatal("unknown id accepted")
	}
}

// Deploy log in Eastern time across the spring-forward
// gap (8 March 2026, 02:00 local skipped): parsed with
// its zone and converted to UTC, the two deploys are
// two minutes apart, not one hour and two minutes.
func TestDSTAndUTCConversion(t *testing.T) {
	east, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tzdata")
	}
	const layout = "2006-01-02 15:04"
	before, _ := time.ParseInLocation(layout, "2026-03-08 01:59", east)
	after, _ := time.ParseInLocation(layout, "2026-03-08 03:01", east)
	at, err := Index([]Event{
		{"deploy:a1", before},
		{"deploy:b2", after},
		{"chat:7", time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d := at["deploy:b2"].Sub(at["deploy:a1"]); d != 2*time.Minute {
		t.Fatalf("gap = %v", d)
	}
	if at["deploy:a1"].Location() != time.UTC {
		t.Fatal("not converted to UTC")
	}
	// The chat message at 07:00 UTC sits between them.
	es := []Entry{
		{Summary: "second deploy", EventIDs: []string{"deploy:b2"}},
		{Summary: "chat", EventIDs: []string{"chat:7"}},
		{Summary: "first deploy", EventIDs: []string{"deploy:a1"}},
	}
	ts, err := Stamp(es, at)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := Order(es, ts)
	want := []string{"first deploy", "chat", "second deploy"}
	for i, w := range want {
		if got[i].Summary != w {
			t.Fatalf("order %d = %s, want %s", i, got[i].Summary, w)
		}
	}
}

// An entry that summarizes events 40 minutes apart is
// stamped with the earliest, by policy, and its IDs
// stay with it so a reader sees the span.
func TestMultiEventEntryTakesEarliest(t *testing.T) {
	t0 := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	at := map[string]time.Time{"k8s:1": t0.Add(40 * time.Minute), "k8s:2": t0}
	es := []Entry{{Summary: "pods restarted twice", Kind: "action",
		EventIDs: []string{"k8s:1", "k8s:2"}}}
	ts, err := Stamp(es, at)
	if err != nil || !ts[0].Equal(t0) || len(es[0].EventIDs) != 2 {
		t.Fatal(ts, err)
	}
}
