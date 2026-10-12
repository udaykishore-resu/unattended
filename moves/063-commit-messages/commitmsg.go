// Package commitmsg drafts a commit message around
// the author's one-line why. Code trims the diff,
// names every file it cut, and falls back to the why
// line alone when the model is slow or down.
//
// Move 63: commit messages and changelogs from diffs.
// The book shows only the prompt; the hook logic it
// describes in prose is implemented minimally here.
package commitmsg

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

// System is the prompt printed in the book. It never
// changes, so the provider can cache it.
const System = `Write a git commit message.
- Subject: imperative, under 60 chars, no period.
- Blank line, then 1-3 short lines of body.
- The body states the WHY using the author's line.
- Do not invent reasons the author did not give.
- Do not list files. Do not describe every change.`

// FileDiff is one file's patch.
type FileDiff struct{ Path, Patch string }

// Generated reports files whose diffs carry bytes but
// no meaning.
func Generated(p string) bool {
	b := path.Base(p)
	return b == "go.sum" || strings.HasSuffix(b, ".pb.go") ||
		strings.HasSuffix(b, "_gen.go")
}

// Trim drops generated files and caps each remaining
// patch at limit bytes. It returns the files it
// truncated, so the hook can say so.
func Trim(fs []FileDiff, limit int) (kept []FileDiff, cut []string) {
	for _, f := range fs {
		if Generated(f.Path) {
			continue
		}
		if len(f.Patch) > limit {
			f.Patch = f.Patch[:limit]
			cut = append(cut, f.Path)
		}
		kept = append(kept, f)
	}
	return kept, cut
}

// ErrNoWhy: the hook refuses to draft without the
// author's reason; the draft would only restate the
// diff.
var ErrNoWhy = errors.New("commitmsg: write one line of why first")

// Timeout is how long the hook waits for the model.
var Timeout = 2 * time.Second

// MaxPatch caps each file's diff, in bytes.
const MaxPatch = 4000

// Draft returns the message to open in the editor and
// whether the model wrote it. On a model error or
// timeout it returns the why line alone. If Trim cut
// any file, code appends a trailer naming it, after
// the model, so a confident draft can't hide that the
// model saw part of the change.
func Draft(ctx context.Context, m llm.Model, why string,
	fs []FileDiff) (string, bool, error) {
	why = strings.TrimSpace(why)
	if why == "" {
		return "", false, ErrNoWhy
	}
	kept, cut := Trim(fs, MaxPatch)
	var b strings.Builder
	fmt.Fprintf(&b, "Author's why: %s\n", why)
	if len(cut) > 0 {
		fmt.Fprintf(&b, "Truncated diffs (partial view): %s\n",
			strings.Join(cut, ", "))
	}
	for _, f := range kept {
		fmt.Fprintf(&b, "--- %s\n%s\n", f.Path, f.Patch)
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	resp, err := m.Complete(ctx, llm.Request{
		System:      System,
		CacheSystem: true,
		Messages:    []llm.Message{{Role: "user", Content: b.String()}},
		MaxTokens:   200,
	})
	msg, ok := strings.TrimSpace(resp.Text), true
	if err != nil || msg == "" {
		msg, ok = why, false
	}
	if len(cut) > 0 {
		msg += "\n\nDiff-Truncated: " + strings.Join(cut, ", ")
	}
	return msg, ok, nil
}
