package handoff

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// MaxReply caps the draft in bytes. Not every
// provider enforces schema length limits, so Go does.
const MaxReply = 1200

var (
	loadRe   = regexp.MustCompile(`^HF-[0-9]{5}$`)
	statuses = []string{"at_shipper", "in_transit",
		"delayed", "delivered"}
)

// Check is business validation, run after decoding
// whatever the provider did with the schema: a load
// ID in Halcyon's format, a known status, an ETA that
// parses (or none), and a non-empty, capped reply.
func (p Packet) Check() error {
	switch {
	case !loadRe.MatchString(p.LoadID):
		return fmt.Errorf("load_id %q: want HF-nnnnn", p.LoadID)
	case !slices.Contains(statuses, p.Status):
		return fmt.Errorf("status %q: unknown", p.Status)
	case strings.TrimSpace(p.Reply) == "":
		return errors.New("draft_reply is empty")
	case len(p.Reply) > MaxReply:
		return fmt.Errorf("draft_reply is %d bytes (max %d)",
			len(p.Reply), MaxReply)
	}
	if p.NewETA != "" {
		if _, err := time.Parse(time.RFC3339, p.NewETA); err != nil {
			return fmt.Errorf("new_eta %q: %v", p.NewETA, err)
		}
	}
	return nil
}

// Load is the current record, read by code.
type Load struct {
	ID     string
	Status string
	ETA    string
}

// FlagLookup is a deterministic uncertainty check:
// the model's Unsure is its own claim, so code adds
// flags of its own. rows is how many loads matched
// the extracted ID in the database.
func FlagLookup(p *Packet, rows int) {
	if rows != 1 {
		addUnsure(p, fmt.Sprintf(
			"load lookup matched %d rows", rows))
	}
}

// FlagChanged re-reads the load when the dispatcher
// opens the packet and flags fields that moved since
// the draft was written.
func FlagChanged(p *Packet, cur Load) []string {
	var changed []string
	if cur.Status != p.Status {
		changed = append(changed, "status")
	}
	if cur.ETA != p.NewETA {
		changed = append(changed, "new_eta")
	}
	if len(changed) > 0 {
		addUnsure(p, "changed since draft: "+
			strings.Join(changed, ", "))
	}
	return changed
}

func addUnsure(p *Packet, s string) {
	if p.Unsure != "" {
		p.Unsure += "; "
	}
	p.Unsure += s
}

// ErrSeeded is returned by the send path for planted
// review-test packets.
var ErrSeeded = errors.New("seeded packet: not sent")

// Sender delivers a reply to a carrier.
type Sender interface {
	Send(ctx context.Context, loadID, body string) error
}

// Send is the only path to a carrier. It refuses
// seeded packets, so a plant a reviewer misses still
// never leaves the building.
func Send(ctx context.Context, s Sender, p Packet) error {
	if p.Seeded {
		return ErrSeeded
	}
	if err := p.Check(); err != nil {
		return err
	}
	return s.Send(ctx, p.LoadID, p.Reply)
}
