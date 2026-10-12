// Package scheduling checks a candidate slot against
// working hours in every attendee's zone. The model
// writes the rules; Go finds the times.
//
// Move 75: scheduling assistant with hard constraints.
package scheduling

import (
	"fmt"
	"time"
)

// Rules are what the model read from the request,
// checked before use (Move 21).
type Rules struct {
	Minutes          int // meeting length
	Earliest, Latest int // local hours, e.g. 9 and 17
	WithinDays       int // how soon: search window
}

// Validate rejects rules the model misread. The
// limits are Halcyon's defaults: at most two hours,
// local hours inside 6 to 20, a window of at most
// three weeks. "A quick 15" read as 150 fails here.
func (r Rules) Validate() error {
	switch {
	case r.Minutes < 15 || r.Minutes > 120:
		return fmt.Errorf("length %d min out of range",
			r.Minutes)
	case r.Earliest < 6 || r.Latest > 20 ||
		r.Earliest >= r.Latest:
		return fmt.Errorf("hours %d-%d out of range",
			r.Earliest, r.Latest)
	case r.WithinDays < 1 || r.WithinDays > 21:
		return fmt.Errorf("window %d days out of range",
			r.WithinDays)
	}
	return nil
}

func inHours(s, e time.Time, r Rules,
	locs []*time.Location) bool {
	for _, loc := range locs {
		ls, le := s.In(loc), e.In(loc)
		if ls.Hour() < r.Earliest {
			return false
		}
		mins := le.Hour()*60 + le.Minute()
		if mins > r.Latest*60 || ls.Day() != le.Day() {
			return false
		}
		wd := ls.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			return false
		}
	}
	return true
}

// Slots walks forward in 15-minute steps and returns
// up to n starts that pass inHours and are free,
// within r.WithinDays of from. Call r.Validate first.
// busy reports whether [s, e) overlaps a busy block.
// Described in prose; minimal.
func Slots(from time.Time, r Rules,
	locs []*time.Location, n int,
	busy func(s, e time.Time) bool) []time.Time {
	var out []time.Time
	s := from.Truncate(15 * time.Minute)
	end := from.AddDate(0, 0, r.WithinDays)
	d := time.Duration(r.Minutes) * time.Minute
	for ; s.Before(end) && len(out) < n; s = s.Add(15 * time.Minute) {
		if inHours(s, s.Add(d), r, locs) && !busy(s, s.Add(d)) {
			out = append(out, s)
		}
	}
	return out
}
