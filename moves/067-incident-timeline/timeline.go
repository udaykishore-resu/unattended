// Package timeline attaches times to model-written
// timeline entries from the event IDs they cite. The
// model never handles a timestamp.
//
// Move 67: auto-draft the incident timeline.
package timeline

import (
	"fmt"
	"slices"
	"time"
)

// Entry is one model-written line with the event
// IDs it came from.
type Entry struct {
	Summary  string   `json:"summary"`
	Kind     string   `json:"kind"` // hypothesis|action|finding
	EventIDs []string `json:"event_ids"`
}

// Stamp gives each entry the time of the earliest
// event it cites. That is the policy for an entry
// that groups several events: it sorts where it
// began, and its IDs show the rest.
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

// Event is one parsed source event. Each source's
// parser converts At to UTC at the edge.
type Event struct {
	ID string
	At time.Time
}

// Index builds the ID -> UTC time map Stamp reads. A
// duplicate ID is a parser bug, so it fails the
// build instead of letting one event overwrite
// another. Described in prose; minimal.
func Index(evs []Event) (map[string]time.Time, error) {
	at := make(map[string]time.Time, len(evs))
	for _, e := range evs {
		if _, dup := at[e.ID]; dup {
			return nil, fmt.Errorf("duplicate event id %q", e.ID)
		}
		at[e.ID] = e.At.UTC()
	}
	return at, nil
}

// Order sorts entries by their stamped times. The
// order the model returned them in is ignored; ties
// keep it.
func Order(es []Entry, ts []time.Time) ([]Entry, []time.Time) {
	idx := make([]int, len(es))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		return ts[a].Compare(ts[b])
	})
	oe := make([]Entry, len(es))
	ot := make([]time.Time, len(es))
	for i, j := range idx {
		oe[i], ot[i] = es[j], ts[j]
	}
	return oe, ot
}
