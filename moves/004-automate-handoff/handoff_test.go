package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

const good = `{
  "load_id": "HF-90412", "status": "delayed",
  "new_eta": "2026-02-14T14:00:00-05:00",
  "draft_reply": "Thanks, noted.",
  "unsure_about": "carrier gave a range; used the late end"}`

func fake(text string) *llm.FakeModel {
	return &llm.FakeModel{Responses: []llm.Response{{Text: text}}}
}

func TestDraftReturnsPacket(t *testing.T) {
	f := fake(good)
	p, err := Draft(context.Background(), f, "sys", "truck late")
	if err != nil {
		t.Fatal(err)
	}
	if p.LoadID != "HF-90412" || p.Unsure == "" {
		t.Fatalf("packet = %+v", p)
	}
	// The request carried the schema; the output is a draft, not an action.
	if !json.Valid(f.Calls[0].Schema) {
		t.Fatal("request schema is not valid JSON")
	}
}

func TestTruncatedOrRefusedIsError(t *testing.T) {
	for _, stop := range []llm.StopReason{
		llm.StopMaxTokens, llm.StopRefusal} {
		f := &llm.FakeModel{Responses: []llm.Response{
			{Text: `{"load_id": "HF-1"`, StopReason: stop}}}
		if _, err := Draft(context.Background(), f, "", "x"); err == nil {
			t.Errorf("%s: want an error before parsing", stop)
		}
	}
}

// Passing a schema is a request, not a guarantee. If
// the provider lets a bad packet through, Check (or
// the decoder) still refuses it.
func TestBadOutputRefusedEvenWithSchema(t *testing.T) {
	cases := map[string]string{
		"not json":      `here is the packet`,
		"wrong type":    `{"load_id": 90412}`,
		"missing load":  `{"status": "delayed", "draft_reply": "ok"}`,
		"carrier ref":   strings.Replace(good, "HF-90412", "PRO 5512", 1),
		"unknown state": strings.Replace(good, `"delayed"`, `"probably fine"`, 1),
		"bad eta":       strings.Replace(good, "2026-02-14T14:00:00-05:00", "Sat 2pm", 1),
		"empty reply":   strings.Replace(good, "Thanks, noted.", "  ", 1),
		"huge reply": strings.Replace(good, "Thanks, noted.",
			strings.Repeat("x", MaxReply+1), 1),
	}
	for name, text := range cases {
		if _, err := Draft(context.Background(), fake(text), "", "x"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestNoETAIsAllowed(t *testing.T) {
	text := strings.Replace(good, "2026-02-14T14:00:00-05:00", "", 1)
	if _, err := Draft(context.Background(), fake(text), "", "x"); err != nil {
		t.Fatal(err)
	}
}

// The model cannot mark a packet seeded or unseeded:
// the field has no JSON name.
func TestModelCannotTouchSeeded(t *testing.T) {
	text := strings.Replace(good, `"status"`, `"Seeded": true, "-": true, "status"`, 1)
	p, err := Draft(context.Background(), fake(text), "", "x")
	if err != nil {
		t.Fatal(err)
	}
	if p.Seeded {
		t.Fatal("model output set Seeded")
	}
}

type outbox struct{ sent []string }

func (o *outbox) Send(_ context.Context, id, body string) error {
	o.sent = append(o.sent, id)
	return nil
}

func TestSendPathRefusesSeeded(t *testing.T) {
	p, err := Draft(context.Background(), fake(good), "", "x")
	if err != nil {
		t.Fatal(err)
	}
	o := &outbox{}
	plant := p
	plant.Seeded = true // a known-bad draft planted for review
	if err := Send(context.Background(), o, plant); !errors.Is(err, ErrSeeded) {
		t.Fatalf("seeded: err = %v, want ErrSeeded", err)
	}
	if err := Send(context.Background(), o, p); err != nil {
		t.Fatal(err)
	}
	if len(o.sent) != 1 {
		t.Fatalf("sent %d, want only the real packet", len(o.sent))
	}
}

func TestFlagLookup(t *testing.T) {
	for rows, want := range map[int]bool{0: true, 1: false, 2: true} {
		p := Packet{}
		FlagLookup(&p, rows)
		if got := p.Unsure != ""; got != want {
			t.Errorf("rows=%d: flagged=%v want %v", rows, got, want)
		}
	}
	p := Packet{Unsure: "model guess"}
	FlagLookup(&p, 2)
	if p.Unsure != "model guess; load lookup matched 2 rows" {
		t.Fatalf("Unsure = %q", p.Unsure)
	}
}

func TestFlagChanged(t *testing.T) {
	p := Packet{Status: "delayed", NewETA: "2026-02-14T14:00:00-05:00"}
	if c := FlagChanged(&p, Load{Status: "delayed",
		ETA: "2026-02-14T14:00:00-05:00"}); len(c) != 0 || p.Unsure != "" {
		t.Fatalf("unchanged load flagged: %v %q", c, p.Unsure)
	}
	c := FlagChanged(&p, Load{Status: "delivered",
		ETA: "2026-02-14T16:00:00-05:00"})
	if strings.Join(c, ",") != "status,new_eta" ||
		!strings.Contains(p.Unsure, "changed since draft") {
		t.Fatalf("changed = %v, Unsure = %q", c, p.Unsure)
	}
}
