package capstone

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // zones resolve the same everywhere
)

// zones maps what carriers write to an IANA zone.
// Carriers write CST in July, so the daylight and
// standard spellings map to the same zone.
var zones = map[string]string{
	"ET": "America/New_York", "EST": "America/New_York",
	"EDT": "America/New_York", "CT": "America/Chicago",
	"CST": "America/Chicago", "CDT": "America/Chicago",
	"MT": "America/Denver", "MST": "America/Denver",
	"MDT": "America/Denver", "PT": "America/Los_Angeles",
	"PST": "America/Los_Angeles",
	"PDT": "America/Los_Angeles",
}

var days = map[string]time.Weekday{
	"sun": time.Sunday, "mon": time.Monday,
	"tue": time.Tuesday, "wed": time.Wednesday,
	"thu": time.Thursday, "fri": time.Friday,
	"sat": time.Saturday,
}

// etaRe reads "<day> <time> [zone]", such as
// "Sat 2pm CT", "Saturday 2:30 p.m." or "Sat 14:00".
var etaRe = regexp.MustCompile(`(?i)^([a-z]{3})[a-z]*\.?,?` +
	`\s+(\d{1,2})(?::(\d{2}))?\s*([ap])?\.?\s*(m\.?)?` +
	`(?:\s+([a-z]{2,3}))?$`)

var ErrETA = errors.New("eta")

// ParseETA turns the time the model copied into a
// timestamp. A zone in the text wins; otherwise the
// delivery city's zone applies. It resolves the
// weekday to the next one on or after the day the
// email was sent, and fails closed on anything it
// cannot read for certain.
func ParseETA(s string, sent time.Time,
	delivery *time.Location) (time.Time, error) {
	m := etaRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return time.Time{}, fmt.Errorf(
			"%w: cannot read %q", ErrETA, s)
	}
	wd, ok := days[strings.ToLower(m[1])]
	if !ok {
		return time.Time{}, fmt.Errorf(
			"%w: no weekday in %q", ErrETA, s)
	}
	h, _ := strconv.Atoi(m[2])
	mi := 0
	if m[3] != "" {
		mi, _ = strconv.Atoi(m[3])
	}
	switch ap := strings.ToLower(m[4]); {
	case ap == "" && m[3] == "":
		return time.Time{}, fmt.Errorf(
			"%w: no am/pm in %q", ErrETA, s)
	case ap != "" && (h < 1 || h > 12):
		return time.Time{}, fmt.Errorf(
			"%w: bad hour in %q", ErrETA, s)
	case ap == "p" && h != 12:
		h += 12
	case ap == "a" && h == 12:
		h = 0
	}
	if h > 23 || mi > 59 {
		return time.Time{}, fmt.Errorf(
			"%w: bad time in %q", ErrETA, s)
	}
	loc := delivery
	if z := strings.ToUpper(m[6]); z != "" {
		name, ok := zones[z]
		if !ok {
			return time.Time{}, fmt.Errorf(
				"%w: unknown zone %q", ErrETA, z)
		}
		loc, _ = time.LoadLocation(name)
	}
	if loc == nil {
		return time.Time{}, fmt.Errorf(
			"%w: no zone for %q", ErrETA, s)
	}
	return resolve(wd, h, mi, sent, loc, s)
}

func resolve(wd time.Weekday, h, mi int,
	sent time.Time, loc *time.Location,
	s string) (time.Time, error) {
	local := sent.In(loc)
	ahead := (int(wd) - int(local.Weekday()) + 7) % 7
	d := local.AddDate(0, 0, ahead)
	t := time.Date(d.Year(), d.Month(), d.Day(),
		h, mi, 0, 0, loc)
	if t.Hour() != h || t.Minute() != mi {
		// time.Date moved it: a spring-forward gap.
		return time.Time{}, fmt.Errorf(
			"%w: %q does not exist in %s",
			ErrETA, s, loc)
	}
	if !t.After(sent) {
		// Same weekday, earlier hour: today is past,
		// and a week out is a guess.
		return time.Time{}, fmt.Errorf(
			"%w: %q is not after the email", ErrETA, s)
	}
	return t, nil
}
