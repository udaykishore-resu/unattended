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
	Usage       llm.Usage      // all four fields
	PromptVer   string
	TraceID     string
}

// Alarms compares a report with recent ones: Seen
// below half its rolling median, or too many
// failures. Described in prose; minimal.
func Alarms(r Report, recent []Report,
	maxFailed int) []string {
	var out []string
	if len(recent) > 0 {
		seen := make([]int, len(recent))
		for i, p := range recent {
			seen[i] = p.Seen
		}
		slices.Sort(seen)
		med := seen[len(seen)/2]
		if r.Seen*2 < med {
			out = append(out, fmt.Sprintf(
				"seen %d, rolling median %d", r.Seen, med))
		}
	}
	if len(r.Failed) > maxFailed {
		out = append(out, fmt.Sprintf(
			"failed %d > %d", len(r.Failed), maxFailed))
	}
	return out
}
