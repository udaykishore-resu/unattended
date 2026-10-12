package golden

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite")

func TestPromptGolden(t *testing.T) {
	ins, _ := filepath.Glob("testdata/*.input")
	for _, in := range ins {
		golden := strings.TrimSuffix(in, ".input") +
			".golden"
		email, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		got := Render(string(email))
		if *update {
			os.WriteFile(golden, []byte(got), 0o644)
		}
		want, _ := os.ReadFile(golden)
		if got != string(want) {
			t.Errorf("%s: prompt changed; run -update",
				golden)
		}
	}
}

// Render is the small test helper the book names: it
// builds the full prompt (system plus user turn) for
// one input. A fixed rule list keeps it deterministic:
// no clock, no random IDs.
func Render(email string) string {
	rules := []string{
		"Pick one label: wismo, claim, invoice, other.",
		"Never guess a load ID.",
	}
	var b strings.Builder
	b.WriteString("You label ops@ email for Halcyon.\n\nRules:\n")
	for i, r := range rules {
		b.WriteString(string(rune('1'+i)) + ". " + r + "\n")
	}
	b.WriteString("\n--- user ---\n<email>\n")
	b.WriteString(email)
	b.WriteString("</email>\n")
	return b.String()
}

// The test must notice a one-word template change.
func TestGoldenCatchesChange(t *testing.T) {
	in, err := os.ReadFile("testdata/spanish.input")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/spanish.golden")
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(Render(string(in)), "Halcyon", "Halcyon Freight", 1)
	if changed == string(want) {
		t.Fatal("a changed prompt matched the golden")
	}
	if n, _ := filepath.Glob("testdata/*.golden"); len(n) != 3 {
		t.Fatalf("want 3 golden files, have %d", len(n))
	}
}
