// Package runreport files one report per scheduled
// run and compares it with recent runs.
//
// Move 60: scheduled agents that file a run report.
package runreport

import (
	"fmt"
	"slices"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

type Report struct {
	Job         string
	Start, End  time.Time
	Seen, Acted int
	Skipped     map[string]int // reason -> count
	Failed      []string       // item IDs
	Pending     []string       // deferred or in review
	Retries     int            // extra attempts, not items
	Usage       llm.Usage      // all four fields
	PromptVer   string
	TraceID     string
}

// Unbalanced reports how far the item buckets are
// from Seen. Every item seen ends in exactly one
// bucket: acted, skipped, failed or pending (left
// for a later run, the DLQ's replay or a person).
// Retries counts extra attempts, so it is not a
// bucket. Zero means the totals reconcile.
func (r Report) Unbalanced() int {
	n := r.Acted + len(r.Failed) + len(r.Pending)
	for _, c := range r.Skipped {
		n += c
	}
	return r.Seen - n
}

// Alarms compares a report with recent ones: buckets
// that don't add up to Seen, Seen below half its
// rolling median, runtime over double its rolling
// median, or too many failures. Described in prose;
// minimal.
func Alarms(r Report, recent []Report,
	maxFailed int) []string {
	var out []string
	if d := r.Unbalanced(); d != 0 {
		out = append(out, fmt.Sprintf(
			"unbalanced: %d items in no bucket", d))
	}
	if len(recent) > 0 {
		seen := make([]int, len(recent))
		took := make([]time.Duration, len(recent))
		for i, p := range recent {
			seen[i] = p.Seen
			took[i] = p.End.Sub(p.Start)
		}
		slices.Sort(seen)
		slices.Sort(took)
		med := seen[len(seen)/2]
		if r.Seen*2 < med {
			out = append(out, fmt.Sprintf(
				"seen %d, rolling median %d", r.Seen, med))
		}
		medT := took[len(took)/2]
		if medT > 0 && r.End.Sub(r.Start) > 2*medT {
			out = append(out, fmt.Sprintf(
				"runtime %s, rolling median %s",
				r.End.Sub(r.Start), medT))
		}
	}
	if len(r.Failed) > maxFailed {
		out = append(out, fmt.Sprintf(
			"failed %d > %d", len(r.Failed), maxFailed))
	}
	return out
}
