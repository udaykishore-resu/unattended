package golden

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The book's code starts here.

var update = flag.Bool("update", false, "rewrite")

func TestPromptGolden(t *testing.T) {
	ins, err := filepath.Glob("testdata/*.input")
	if err != nil || len(ins) == 0 {
		t.Fatal("no inputs", err)
	}
	for _, in := range ins {
		golden := strings.TrimSuffix(in, ".input") +
			".golden"
		email, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		got := Render(string(email))
		if *update {
			err := os.WriteFile(golden, []byte(got), 0o644)
			if err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Error(golden, "prompt changed; run -update")
		}
	}
}

// The book's code ends here.

// Render is the small test helper the book names: it
// builds the full prompt (system plus user turn) for
// one input. A fixed rule list keeps it deterministic:
// no clock, no random IDs. Sender text is defanged as
// in Move 10, so the closing-tag golden shows the
// neutralised tag the model will see.
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
	b.WriteString(defang(email))
	b.WriteString("</email>\n")
	return b.String()
}

var fenceTag = regexp.MustCompile(`(?i)<\s*/?\s*email`)

// defang is a trimmed copy of Move 10's helper.
func defang(s string) string {
	return fenceTag.ReplaceAllStringFunc(s,
		func(m string) string { return "&lt;" + m[1:] })
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
