package skeleton

import (
	"strings"
	"testing"
)

func TestOrderAndNumbering(t *testing.T) {
	s := Skeleton{
		Role:     "You label ops@ email for Halcyon dispatch.",
		Rules:    []string{"Pick one label.", "Never guess a load ID."},
		Examples: "<example>...</example>\n",
	}
	got := s.System()
	want := "You label ops@ email for Halcyon dispatch.\n\nRules:\n" +
		"1. Pick one label.\n2. Never guess a load ID.\n" +
		"\nExamples:\n<example>...</example>\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// The stable part (role, rules, examples) never
// contains the input, so it caches as a prefix.
func TestInputStaysOutOfSystem(t *testing.T) {
	s := Skeleton{Role: "r", Rules: []string{"x"}}
	if strings.Contains(s.System(), "CUSTOMER TEXT") {
		t.Fatal("input leaked into system")
	}
	if u := s.User("CUSTOMER TEXT"); !strings.HasPrefix(u, "<input>") {
		t.Fatalf("user turn not fenced: %q", u)
	}
}

func TestEveryPromptHasRulesHeader(t *testing.T) {
	if !strings.Contains((Skeleton{Role: "r"}).System(), "Rules:") {
		t.Fatal("missing Rules: header")
	}
}
