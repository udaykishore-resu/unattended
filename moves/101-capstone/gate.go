package capstone

import (
	"strings"
	"time"
)

// Load is the shipments-table row the gate checks
// a proposal against.
type Load struct {
	ID            string
	CarrierDomain string // the assigned carrier's
	InTransit     bool
	ETA           time.Time // current
	Delivery      *time.Location
}

// MaxShift is Halcyon's L4 limit on how far one
// email may move an ETA without a person.
const MaxShift = 48 * time.Hour

// Gate decides whether a validated update may be
// applied without a person. Any "no" sends the email
// to the review queue as an L2 draft. Gate is pure:
// a rejection is a decision, so it is never retried.
func Gate(e Email, c Checked, l Load,
	found bool) (ok bool, why string) {
	switch {
	case c.Abstained():
		return false, "model abstained"
	case !found:
		return false, "unknown load " + *c.LoadID
	case !e.AuthAligned:
		return false, "sender failed SPF/DKIM/DMARC"
	case !strings.EqualFold(e.FromDomain,
		l.CarrierDomain):
		return false, "sender is not the assigned " +
			"carrier for " + l.ID
	case !l.InTransit:
		return false, l.ID + " is not in transit"
	case absDur(c.ETA.Sub(l.ETA)) >= MaxShift:
		return false, "shift of 48h or more"
	}
	return true, ""
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
