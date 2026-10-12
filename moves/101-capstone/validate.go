package capstone

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Email is one message from the ops@ webhook, with
// the authentication results the mail server added.
type Email struct {
	MessageID  string
	FromDomain string
	// SPF, DKIM and DMARC passed, with alignment.
	AuthAligned   bool
	Subject, Body string
	Sent          time.Time
}

// Update is one item of the model's output.
type Update struct {
	LoadID   *string `json:"load_id"`
	Status   string  `json:"status"`
	NewETA   *string `json:"new_eta"`
	Reason   *string `json:"reason"`
	Evidence string  `json:"evidence"`
}

// Checked is an Update that passed Validate. ETA is
// zero when the model abstained.
type Checked struct {
	Update
	ETA time.Time
}

// Abstained reports a "not_present" answer.
func (c Checked) Abstained() bool {
	return c.Status == "not_present"
}

var loadIDRe = regexp.MustCompile(`^HF-[0-9]{5}$`)

// ValidationError lists every problem at once, so
// the one repair call can fix them all.
type ValidationError struct{ Problems []string }

func (e *ValidationError) Error() string {
	return "validation: " +
		strings.Join(e.Problems, "; ")
}

// ZoneFor finds the delivery city's zone for a load.
type ZoneFor func(loadID string) *time.Location

// Validate decodes the output strictly and enforces
// in Go what the prompt asks for: the load ID
// pattern, the ID appearing in the email, the reason
// enum, evidence quoted from the email, and an ETA
// Go can parse in the right zone.
func Validate(raw string, e Email,
	zone ZoneFor) ([]Checked, error) {
	var out struct {
		Updates []Update `json:"updates"`
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return nil, &ValidationError{[]string{
			"decode: " + err.Error()}}
	}
	if dec.More() {
		return nil, &ValidationError{[]string{
			"decode: trailing data after the JSON"}}
	}
	if len(out.Updates) == 0 {
		return nil, &ValidationError{[]string{
			"updates: empty list"}}
	}
	text := e.Subject + "\n" + e.Body
	var probs []string
	var res []Checked
	for i, u := range out.Updates {
		c, p := check(i, u, text, e.Sent, zone)
		probs = append(probs, p...)
		res = append(res, c)
	}
	if len(probs) > 0 {
		return nil, &ValidationError{probs}
	}
	return res, nil
}

func check(i int, u Update, text string,
	sent time.Time, zone ZoneFor) (Checked, []string) {
	c := Checked{Update: u}
	var p []string
	bad := func(f string, a ...any) {
		p = append(p, fmt.Sprintf("updates[%d].", i)+
			fmt.Sprintf(f, a...))
	}
	switch u.Status {
	case "not_present":
		return c, nil // abstention is an answer
	case "delayed":
	default:
		bad("status: %q is not an allowed value",
			u.Status)
		return c, p
	}
	id := deref(u.LoadID)
	switch {
	case !loadIDRe.MatchString(id):
		bad("load_id: %q is not HF- plus five digits",
			id)
	case !strings.Contains(text, id):
		bad("load_id: %s does not appear in the email",
			id)
	}
	if r := deref(u.Reason); !slices.Contains(Reasons, r) {
		bad("reason: %q is not in the enum", r)
	}
	if ev := strings.TrimSpace(u.Evidence); ev == "" ||
		!strings.Contains(squash(text), squash(ev)) {
		bad("evidence: not a quote from the email")
	}
	var loc = zone(id)
	eta, err := ParseETA(deref(u.NewETA), sent, loc)
	if err != nil {
		bad("new_eta: %v", err)
	}
	c.ETA = eta
	return c, p
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// squash folds runs of whitespace, so a quote that
// crossed a line break still matches.
func squash(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
