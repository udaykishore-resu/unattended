package chunking

import (
	"reflect"
	"strings"
	"testing"
)

func paths(secs []Section) []string {
	var out []string
	for _, s := range secs {
		out = append(out, strings.Join(s.Path, " > "))
	}
	return out
}

// Text before the first heading is kept, with an
// empty trail.
func TestPreambleKept(t *testing.T) {
	got := SplitMarkdown("Intro line.\n# A\nbody\n")
	if len(got) != 2 || got[0].Path != nil ||
		!strings.Contains(got[0].Body, "Intro line.") {
		t.Fatalf("got %+v", got)
	}
}

// Skipped levels: a ### under # and then a ## must
// pop back to the right parent, and a later ###
// under the ## must not keep the earlier ###.
func TestSkippedLevelsAndShallowerHeadings(t *testing.T) {
	doc := "# A\na\n### C\nc\n#### D\nd\n### E\ne\n" +
		"## B\nb\n### F\nf\n# G\ng\n"
	want := []string{"A", "A > C", "A > C > D", "A > E",
		"A > B", "A > B > F", "G"}
	if got := paths(SplitMarkdown(doc)); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths:\n%q\nwant:\n%q", got, want)
	}
}

// Empty headings and seven-# lines are body text,
// never an empty trail entry.
func TestEmptyAndTooDeepHeadings(t *testing.T) {
	doc := "# A\n#\n##   \n####### seven\nbody\n"
	got := SplitMarkdown(doc)
	if len(got) != 1 || !reflect.DeepEqual(got[0].Path, []string{"A"}) {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got[0].Body, "####### seven") {
		t.Fatalf("seven-# line lost: %q", got[0].Body)
	}
	for _, s := range got {
		for _, p := range s.Path {
			if p == "" {
				t.Fatalf("empty trail entry in %+v", got)
			}
		}
	}
}

// A shell comment inside a fence is not a heading.
func TestHeadingInsideFence(t *testing.T) {
	doc := "# Runbook\n## Restart\nSteps:\n```sh\n" +
		"# restart the pod\nkubectl rollout restart\n```\n" +
		"~~~\n## not a heading\n~~~\n## After\ndone\n"
	got := SplitMarkdown(doc)
	want := []string{"Runbook > Restart", "Runbook > After"}
	if p := paths(got); !reflect.DeepEqual(p, want) {
		t.Fatalf("paths %q, want %q", p, want)
	}
	if !strings.Contains(got[0].Body, "# restart the pod") ||
		!strings.Contains(got[0].Body, "## not a heading") {
		t.Fatalf("fenced lines left the body: %q", got[0].Body)
	}
}

// Sections holding only whitespace are dropped; a
// heading with no body survives only in its children.
func TestWhitespaceOnlySectionsDropped(t *testing.T) {
	doc := "   \n\n# A\n \t\n## B\nreal text\n## C\n\n\n"
	got := SplitMarkdown(doc)
	if p := paths(got); !reflect.DeepEqual(p, []string{"A > B"}) {
		t.Fatalf("paths %q", p)
	}
}

func TestIsFence(t *testing.T) {
	cases := map[string]bool{"```": true, "```go": true,
		"   ~~~": true, "    ```": false, "``": false,
		"text ```": false}
	for in, want := range cases {
		if got := isFence(in); got != want {
			t.Errorf("isFence(%q) = %v, want %v", in, got, want)
		}
	}
}
