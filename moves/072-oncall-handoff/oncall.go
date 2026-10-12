// Package oncall lists every silence or override
// that ends during the next shift. Computed in code,
// never by the model.
//
// Move 72: on-call handoff summaries.
package oncall

import (
	"sort"
	"time"
)

// Silence is an alert silence, snooze or temporary
// override with an end time.
type Silence struct {
	Alert, By string
	Ends      time.Time
}

func ExpiringSoon(ss []Silence, from time.Time,
	shift time.Duration) []Silence {
	var out []Silence
	until := from.Add(shift)
	for _, s := range ss {
		if s.Ends.After(from) &&
			s.Ends.Before(until) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Ends.Before(out[j].Ends)
	})
	return out
}
