package promptlint

import (
	"slices"
	"strings"
	"testing"
)

// One table case per check, so loosening a pattern
// shows what broke.
func TestLint(t *testing.T) {
	cases := []struct {
		name, prompt, want string
	}{
		{"vague verb", "Handle refund requests.", "1: vague verb Handle"},
		{"unfenced slot", "Summarize: {{.Body}}", "1: unfenced variable"},
		{"conflict",
			strings.Repeat("x\n", 6) + "Always reply in English.\n" +
				strings.Repeat("x\n", 11) +
				"Never reply in English to Quebec senders.",
			"19: never reply in conflicts with line 7"},
	}
	for _, c := range cases {
		got := Lint(c.prompt)
		if !slices.Contains(got, c.want) {
			t.Errorf("%s: want %q in %q", c.name, c.want, got)
		}
	}
}

func TestFencedSlotPasses(t *testing.T) {
	p := "Summarize the email.\n<email>\n{{.Body}}\n</email>"
	if got := Lint(p); len(got) != 0 {
		t.Fatalf("clean prompt flagged: %q", got)
	}
}

// Crude on purpose: different wording is not caught.
func TestCrudeByDesign(t *testing.T) {
	p := "Respond only in French.\nAlways reply in English."
	if got := Lint(p); len(got) != 0 {
		t.Fatalf("unexpected findings: %q", got)
	}
}
