package repurpose

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
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

// Models often write curly quotes. They must not
// slip past the check.
func TestInventedCurlyQuotes(t *testing.T) {
	piece := "As Theo said, “we paged on every disk at eighty percent.” " +
		"And “half our alerts fired on things nobody could act on.”"
	got := Invented(piece, transcript)
	if !slices.Equal(got, []string{"we paged on every disk at eighty percent."}) {
		t.Fatalf("got %q", got)
	}
}

// Stage two sees only the approved outline and the
// samples, never the raw transcript.
func TestStageTwoSeesOnlyTheOutline(t *testing.T) {
	outline := "Claim: page only on what a person can act on.\n1. Count pages first."
	f := &llm.FakeModel{Responses: []llm.Response{{
		Text: `Theo's advice: "Alert fatigue is the silent killer of on-call."`}}}
	blog := Format{Name: "blog", System: "Write a blog post for SREs.", Public: true}
	piece, bad, err := Draft(context.Background(), f, blog, outline,
		[]string{"I like short sentences."}, transcript)
	if err != nil || piece == "" {
		t.Fatal(err)
	}
	req := f.Calls[0]
	all := req.System + req.Messages[0].Content
	if strings.Contains(all, "nobody could act on") {
		t.Fatal("transcript leaked into stage two")
	}
	if !strings.Contains(all, outline) || req.System != blog.System {
		t.Fatalf("request = %+v", req)
	}
	if len(bad) != 1 {
		t.Fatalf("invented quote not flagged: %q", bad)
	}
}

// The limit, pinned: a paraphrase that puts a claim
// in the speaker's mouth without quote marks passes.
// That is the reviewer's job.
func TestParaphraseIsNotCaught(t *testing.T) {
	piece := "Theo says alert fatigue is the silent killer of on-call."
	if got := Invented(piece, transcript); len(got) != 0 {
		t.Fatalf("got %q; update the book", got)
	}
}
