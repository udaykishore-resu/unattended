package chunking

import (
	"reflect"
	"strings"
	"testing"
)

const contract = `# Carrier Agreement
Between Halcyon and RDX.
## Detention
### Rates
Detention is billed at $75 per hour after two free hours.
### Disputes
Send disputes within 30 days.
## Fuel
Fuel follows the weekly index.
`

func TestTrailAndWholeIdeas(t *testing.T) {
	got := SplitMarkdown(contract)
	var paths []string
	for _, s := range got {
		paths = append(paths, strings.Join(s.Path, " > "))
	}
	want := []string{
		"Carrier Agreement",
		"Carrier Agreement > Detention > Rates",
		"Carrier Agreement > Detention > Disputes",
		"Carrier Agreement > Fuel",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths:\n%q\nwant:\n%q", paths, want)
	}
	// The rate is never cut away from its sentence.
	if !strings.Contains(got[1].Body, "Detention is billed at $75 per hour") {
		t.Fatalf("rates body = %q", got[1].Body)
	}
}

func TestHeadingLevel(t *testing.T) {
	cases := map[string]int{"# A": 1, "### C": 3, "####### G": 0,
		"#nospace": 0, "plain": 0, "#": 0}
	for in, want := range cases {
		if got := headingLevel(in); got != want {
			t.Errorf("headingLevel(%q) = %d, want %d", in, got, want)
		}
	}
}

// Sibling sections never share a trail slice.
func TestPathsAreIndependent(t *testing.T) {
	got := SplitMarkdown("# A\nx\n## B\ny\n## C\nz\n")
	if got[1].Path[1] != "B" || got[2].Path[1] != "C" {
		t.Fatalf("paths aliased: %+v", got)
	}
}
