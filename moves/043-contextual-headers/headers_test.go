package headers

import (
	"strings"
	"testing"
	"time"
)

func TestHeader(t *testing.T) {
	c := Chunk{ID: "c-17", DocTitle: "Brightline Foods Agreement",
		Text:    "Two free hours, then $75 per hour, billed in 15-minute steps.",
		Path:    []string{"Detention", "Unloading"},
		Updated: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	want := "[Brightline Foods Agreement > Detention > Unloading | updated 2026-02-01]"
	if got := Header(c); got != want {
		t.Fatalf("got %s", got)
	}
	idx := IndexText(c, "Unloading detention terms for Brightline Foods.")
	// "Brightline unloading detention" now has words to match.
	for _, w := range []string{"Brightline", "Unloading", "Detention"} {
		if !strings.Contains(idx, w) {
			t.Errorf("index text lacks %q", w)
		}
	}
	if c.Text != "Two free hours, then $75 per hour, billed in 15-minute steps." {
		t.Fatal("chunk body must stay unchanged; users see only it")
	}
	if !strings.Contains(situatePrompt, "Do not add facts") {
		t.Fatal("keep the no-new-facts line")
	}
}
