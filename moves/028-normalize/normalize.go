// Package normalize converts raw text the model
// copied into dates, in Go, where tests can reach.
//
// Move 28: normalize dates, currency and units
// outside the model.
package normalize

import (
	"fmt"
	"strings"
	"time"
)

var layouts = []string{
	"2006-01-02", "01/02/2006", "1/2/2006",
	"Jan 2, 2006", "January 2, 2006", "2 Jan 2006",
}

func ParseDate(raw string,
	loc *time.Location) (time.Time, error) {
	s := strings.TrimSpace(raw)
	for _, l := range layouts {
		t, err := time.ParseInLocation(l, s, loc)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"unparsed date %q", raw)
}
