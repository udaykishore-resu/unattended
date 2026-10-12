// Package repair validates a tool-call record in Go,
// sends one repair, then rejects.
//
// Move 21: validate, repair once, then reject.
package repair

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

// ErrRejected marks a record that failed after its
// one repair. The caller routes it to a person.
var ErrRejected = errors.New("rejected after repair")

// Extract makes the first call, then runs the
// book's fragment: one repair, then reject.
func Extract(ctx context.Context, m llm.Model,
	req llm.Request) (Tender, error) {
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return Tender{}, err
	}
	// --- the book's code starts here ---
	call, err := oneCall(resp, "save_tender")
	if err != nil {
		return Tender{}, err // prose reply: review
	}
	t, probs := parse(call.Input)
	if len(probs) == 0 {
		return t, nil
	}
	req.Messages = append(req.Messages,
		llm.Message{Role: "assistant",
			Content: resp.Text, ToolCalls: resp.ToolCalls},
		llm.Message{Role: "tool", ToolCallID: call.ID,
			Content: repairNote(probs)})
	if resp, err = m.Complete(ctx, req); err != nil {
		return Tender{}, err
	}
	if call, err = oneCall(resp, "save_tender"); err != nil {
		return Tender{}, err
	}
	if t, probs = parse(call.Input); len(probs) > 0 {
		return Tender{}, fmt.Errorf("%w: %s",
			ErrRejected, strings.Join(probs, "; "))
	}
	return t, nil
}

// Tender is the record from Move 20. Pointers let
// the model answer null (Move 29).
type Tender struct {
	ShipperRef string  `json:"shipper_ref"`
	OriginZip  *string `json:"origin_zip"`
	DestZip    *string `json:"dest_zip"`
	PickupDate string  `json:"pickup_date"`
	WeightLb   int     `json:"weight_lb"`
}

var ErrNoToolCall = errors.New("no tool call")

// oneCall is from Move 20.
func oneCall(r llm.Response, name string) (
	llm.ToolCall, error) {
	if len(r.ToolCalls) != 1 ||
		r.ToolCalls[0].Name != name {
		return llm.ToolCall{}, ErrNoToolCall
	}
	return r.ToolCalls[0], nil
}

var zip5 = regexp.MustCompile(`^[0-9]{5}$`)
var zip9 = regexp.MustCompile(`^([0-9]{5})-[0-9]{4}$`)

// parse decodes strictly and runs every rule on the
// whole record, one problem string per failure.
// Described in prose; minimal implementation.
func parse(in json.RawMessage) (Tender, []string) {
	var t Tender
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, []string{"decode: " + err.Error()}
	}
	var probs []string
	for name, z := range map[string]*string{
		"origin_zip": t.OriginZip, "dest_zip": t.DestZip} {
		if z == nil {
			continue // null is an answer, not a failure
		}
		// Fix what code can fix before spending a call.
		if m := zip9.FindStringSubmatch(*z); m != nil {
			*z = m[1]
		}
		if !zip5.MatchString(*z) {
			probs = append(probs, fmt.Sprintf(
				"%s: %q is not five digits", name, *z))
		}
	}
	if t.WeightLb < 1 || t.WeightLb > 80000 {
		probs = append(probs, fmt.Sprintf(
			"weight_lb: %d is outside 1-80000", t.WeightLb))
	}
	return t, probs
}

// repairNote joins the problems and ends with the
// escape hatch. Described in prose; minimal.
func repairNote(probs []string) string {
	return "These fields failed validation:\n- " +
		strings.Join(probs, "\n- ") +
		"\nCall save_tender again with fixed values. " +
		"If a value is not in the email, use null."
}
