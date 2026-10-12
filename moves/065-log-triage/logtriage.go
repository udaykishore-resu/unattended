// Package logtriage masks values in log lines so
// lines from the same log call collapse into one
// template, then counts in code.
//
// Move 65: log triage: cluster first, summarize
// second.
package logtriage

import (
	"regexp"
	"sort"
)

var masks = []struct {
	re  *regexp.Regexp
	tok string
}{
	{regexp.MustCompile(`"[^"]*"`), `"<s>"`},
	{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f-]{27}`),
		"<uuid>"},
	{regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`),
		"<ip>"},
	{regexp.MustCompile(`\b[A-Z]{2,4}-\d+\b`), "<id>"},
	{regexp.MustCompile(`\d+(\.\d+)?`), "<n>"},
}

func Template(line string) string {
	for _, m := range masks {
		line = m.re.ReplaceAllString(line, m.tok)
	}
	return line
}

// Cluster is one template with its count and up to
// three samples.
type Cluster struct {
	Template string
	Count    int
	Samples  []string
}

// Clusters groups by template, counts, keeps three
// samples per cluster, and sorts by count. The book
// describes this in prose; minimal implementation.
func Clusters(lines []string) []Cluster {
	by := map[string]*Cluster{}
	for _, ln := range lines {
		t := Template(ln)
		c := by[t]
		if c == nil {
			c = &Cluster{Template: t}
			by[t] = c
		}
		c.Count++
		if len(c.Samples) < 3 {
			c.Samples = append(c.Samples, ln)
		}
	}
	out := make([]Cluster, 0, len(by))
	for _, c := range by {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Template < out[j].Template
	})
	return out
}

// New returns templates seen now but not in the
// baseline hour: often the one that matters.
func New(now, baseline []Cluster) []Cluster {
	old := map[string]bool{}
	for _, c := range baseline {
		old[c.Template] = true
	}
	var out []Cluster
	for _, c := range now {
		if !old[c.Template] {
			out = append(out, c)
		}
	}
	return out
}
