// Package normalize converts raw text the model
// copied into dates, cents and pounds, in Go, where
// tests can reach. Ambiguous or unsupported input
// is an error: nothing falls back to a guess.
//
// Move 28: normalize dates, currency and units
// outside the model.
package normalize

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// numeric holds the only all-number layouts, chosen
// by the day order stored on the shipper record.
var numeric = map[string][]string{
	"US": {"1/2/2006"}, // month first
	"EU": {"2/1/2006"}, // day first
}

// named layouts cannot be misread either way.
var named = []string{"2006-01-02",
	"Jan 2, 2006", "January 2, 2006", "2 Jan 2006"}

func ParseDate(raw, order string,
	loc *time.Location) (time.Time, error) {
	num, ok := numeric[order]
	if !ok {
		return time.Time{}, fmt.Errorf(
			"no day order %q on shipper", order)
	}
	s := strings.TrimSpace(raw)
	for _, l := range slices.Concat(num, named) {
		t, err := time.ParseInLocation(l, s, loc)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"unparsed date %q", raw)
}
