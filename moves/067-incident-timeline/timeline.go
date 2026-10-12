// Package timeline attaches times to model-written
// timeline entries from the event IDs they cite. The
// model never handles a timestamp.
//
// Move 67: auto-draft the incident timeline.
package timeline

import (
	"fmt"
	"time"
)

// Entry is one model-written line with the event
// IDs it came from.
type Entry struct {
	Summary  string   `json:"summary"`
	Kind     string   `json:"kind"` // hypothesis|action|finding
	EventIDs []string `json:"event_ids"`
}

func Stamp(es []Entry, at map[string]time.Time) (
	[]time.Time, error) {
	out := make([]time.Time, len(es))
	for i, e := range es {
		if len(e.EventIDs) == 0 {
			return nil, fmt.Errorf("entry %d: no source", i)
		}
		for _, id := range e.EventIDs {
			t, ok := at[id]
			if !ok {
				return nil, fmt.Errorf("bad id %q", id)
			}
			if out[i].IsZero() || t.Before(out[i]) {
				out[i] = t
			}
		}
	}
	return out, nil
}
