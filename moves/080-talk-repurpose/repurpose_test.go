package repurpose

import (
	"slices"
	"testing"
)

const transcript = `So the first thing we did was count pages.
We found that half our alerts
fired on things nobody could act on.`

func TestInvented(t *testing.T) {
	piece := `Theo put it plainly: "half our alerts fired on things nobody could act on." ` +
		`He added, "Alert fatigue is the silent killer of on-call." ` +
		`Short quotes like "count pages" are skipped.`
	got := Invented(piece, transcript)
	if !slices.Equal(got, []string{"Alert fatigue is the silent killer of on-call."}) {
		t.Fatalf("got %q", got)
	}
}
