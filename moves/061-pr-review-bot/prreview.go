// Package prreview keeps only review findings that
// are on an added line, of an allowed kind, rated
// high, and quote the exact code.
//
// Move 61: PR review bot that only comments when
// it's sure.
package prreview

import (
	"context"
	"errors"
	"slices"
	"strings"
)

type Finding struct {
	File, Kind, Evidence string
	Line                 int
	Confidence           string // high|medium|low
}

type Added map[string]map[int]string

func Keep(fs []Finding, diff Added,
	kinds map[string]bool) []Finding {
	var out []Finding
	for _, f := range fs {
		text, ok := diff[f.File][f.Line]
		if ok && kinds[f.Kind] &&
			f.Confidence == "high" &&
			f.Evidence != "" &&
			strings.Contains(text, f.Evidence) {
			out = append(out, f)
		}
	}
	return out
}

// Review is one posted review, pinned to the commit
// its findings were computed against. The code host
// anchors each comment to CommitID, so a later push
// marks it outdated instead of moving it.
type Review struct {
	CommitID string
	Findings []Finding
}

// ErrStale means the PR head moved while the bot ran.
var ErrStale = errors.New("prreview: head moved, run dropped")

// Post sends at most limit findings, worst kind first,
// as one review pinned to the reviewed SHA. It drops
// the run if the head changed before posting. The
// book describes this in prose; minimal version.
func Post(ctx context.Context, reviewed string,
	fs []Finding, order []string, limit int,
	head func(context.Context) (string, error),
	send func(context.Context, Review) error) error {
	if len(fs) == 0 {
		return nil // silence is the default
	}
	h, err := head(ctx)
	if err != nil {
		return err
	}
	if h != reviewed {
		return ErrStale
	}
	rank := map[string]int{}
	for i, k := range order {
		rank[k] = i + 1
	}
	fs = slices.Clone(fs)
	slices.SortStableFunc(fs, func(a, b Finding) int {
		return rank[a.Kind] - rank[b.Kind]
	})
	if len(fs) > limit {
		fs = fs[:limit]
	}
	return send(ctx, Review{CommitID: reviewed, Findings: fs})
}

// DropMoved removes added lines whose text, trimmed,
// matches a line the same PR removed: a pure move is
// not new code. The book's first "Breaks when" fix.
func DropMoved(diff Added, removed []string) Added {
	gone := map[string]bool{}
	for _, r := range removed {
		gone[strings.TrimSpace(r)] = true
	}
	out := Added{}
	for file, lines := range diff {
		for n, text := range lines {
			t := strings.TrimSpace(text)
			if t != "" && gone[t] {
				continue
			}
			if out[file] == nil {
				out[file] = map[int]string{}
			}
			out[file][n] = text
		}
	}
	return out
}
