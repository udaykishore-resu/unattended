package briefs

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Source is one gathered page or record. Key is the
// company identifier the brief is about: a domain or
// a registry number, never just a name.
type Source struct {
	URL, Key    string
	Title, Text string
	Fetched     time.Time
	Err         error // fetch failed or timed out
}

// Limits are the gather step's filters. The values
// are starting defaults, not rules from anywhere.
type Limits struct {
	MinChars int           // shorter is a stub or wall
	MaxAge   time.Duration // older fetches are stale
}

var loginWords = regexp.MustCompile(
	`(?i)\b(log ?in|sign ?in|access denied|captcha)\b`)

// Gather keeps sources that are about this key, were
// fetched without error, recently, and look like real
// pages. Everything dropped is listed with a reason,
// so the brief can say what it skipped.
func Gather(srcs []Source, key string, now time.Time,
	lim Limits) (keep []Source, skipped []string) {
	for _, s := range srcs {
		why := ""
		switch {
		case s.Err != nil:
			why = "fetch failed: " + s.Err.Error()
		case s.Key != key:
			why = "different company key " + s.Key
		case now.Sub(s.Fetched) > lim.MaxAge:
			why = "stale fetch " + s.Fetched.Format(time.DateOnly)
		case len(strings.TrimSpace(s.Text)) < lim.MinChars:
			why = "too short"
		case loginWords.MatchString(s.Title):
			why = "login page"
		}
		if why != "" {
			skipped = append(skipped, s.URL+": "+why)
			continue
		}
		keep = append(keep, s)
	}
	return keep, skipped
}

// Header is printed at the top of every brief: the
// company key, each source with its id and fetch date,
// and what was skipped. Ids match the [Sn] tags.
func Header(key string, keep []Source,
	skipped []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Company key: %s\n", key)
	for i, s := range keep {
		fmt.Fprintf(&b, "[S%d] %s (fetched %s)\n", i+1,
			s.URL, s.Fetched.Format(time.DateOnly))
	}
	for _, s := range skipped {
		fmt.Fprintf(&b, "Skipped: %s\n", s)
	}
	return b.String()
}

var sentence = regexp.MustCompile(`(?s).*?[.!?](\s+|$)`)

// DropUntagged removes sentences that carry no [Sn]
// tag. Sentence splitting is a heuristic (it splits
// after "Inc." too); a split only ever drops text, it
// never keeps an untagged claim.
func DropUntagged(brief string) string {
	var out []string
	rest := brief
	for rest != "" {
		loc := sentence.FindStringIndex(rest)
		end := len(rest)
		if loc != nil {
			end = loc[1]
		}
		s := strings.TrimSpace(rest[:end])
		if cite.MatchString(s) {
			out = append(out, s)
		}
		rest = rest[end:]
	}
	return strings.Join(out, " ")
}
