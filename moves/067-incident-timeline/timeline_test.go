package timeline

import (
	"testing"
	"time"
)

func TestStamp(t *testing.T) {
	east, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tzdata")
	}
	// Each parser converts to UTC at the edge.
	at := map[string]time.Time{
		"chat:1712":     time.Date(2026, 3, 9, 14, 12, 0, 0, time.UTC),
		"deploy:a41f9c": time.Date(2026, 3, 9, 9, 51, 0, 0, east).UTC(), // 13:51 UTC
		"chat:1730":     time.Date(2026, 3, 9, 14, 30, 0, 0, time.UTC),
	}
	es := []Entry{
		{Summary: "deploy region-filter", Kind: "action", EventIDs: []string{"deploy:a41f9c"}},
		{Summary: "failover", Kind: "action", EventIDs: []string{"chat:1730", "chat:1712"}},
	}
	got, err := Stamp(es, at)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Hour() != 13 || got[0].Minute() != 51 {
		t.Errorf("deploy at %v, want 13:51 UTC", got[0])
	}
	// An entry citing several events takes the earliest.
	if !got[1].Equal(at["chat:1712"]) {
		t.Errorf("failover at %v", got[1])
	}
}

func TestMadeUpOrMissingSourceFails(t *testing.T) {
	at := map[string]time.Time{"chat:1": time.Now()}
	if _, err := Stamp([]Entry{{Summary: "x"}}, at); err == nil {
		t.Error("entry with no source passed")
	}
	if _, err := Stamp([]Entry{{EventIDs: []string{"chat:999"}}}, at); err == nil {
		t.Error("made-up ID passed")
	}
}
