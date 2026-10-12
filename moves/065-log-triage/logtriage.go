// Package logtriage masks values in log lines so
// lines from the same log call collapse into one
// template, then counts in code.
//
// Move 65: log triage: cluster first, summarize
// second.
package logtriage

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

var masks = []struct {
	re  *regexp.Regexp
	tok string
}{
	{regexp.MustCompile(`"[^"]*"`), `"<s>"`},
	{regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f-]{27}`),
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

// Prompt builds the model input from the top
// clusters. The masks exist for grouping, not
// redaction: samples are raw lines, so every sample
// passes through redact before it is written here.
// Described in prose; minimal implementation.
func Prompt(cs []Cluster, redact func(string) string) string {
	var b strings.Builder
	b.WriteString("Which three clusters most likely explain an " +
		"outage, and which look like everyday noise? Do not " +
		"guess at causes the samples do not show.\n")
	for i, c := range cs {
		fmt.Fprintf(&b, "C%d count=%d template=%s\n", i+1, c.Count,
			redact(c.Template))
		for _, s := range c.Samples {
			fmt.Fprintf(&b, "  sample: %s\n", redact(s))
		}
	}
	return b.String()
}

// Ask sends the top n clusters to the model, redacted.
func Ask(ctx context.Context, m llm.Model, cs []Cluster, n int,
	redact func(string) string) (string, error) {
	if len(cs) > n {
		cs = cs[:n]
	}
	resp, err := m.Complete(ctx, llm.Request{
		Messages:  []llm.Message{{Role: "user", Content: Prompt(cs, redact)}},
		MaxTokens: 400,
	})
	return resp.Text, err
}
