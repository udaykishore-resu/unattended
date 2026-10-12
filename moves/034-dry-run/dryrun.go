// Package dryrun splits a write tool into a plan
// half that reads and computes, and an apply half
// that writes.
//
// Move 34: dry-run mode for every write tool.
package dryrun

import (
	"context"
	"fmt"
)

func planSetStatus(ctx context.Context, db Store,
	in SetStatusIn) (Plan, error) {
	l, err := db.GetLoad(ctx, in.LoadID)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Tool: "set_status", Target: l.ID,
		Changes: map[string][2]any{
			"status": {l.Status, in.Status}}}
	if l.Status == in.Status {
		p.Notes = append(p.Notes, "no-op: same status")
	}
	return p, nil
}

// Types and the apply half below are implied by the
// book's prose; minimal implementations.

type Load struct{ ID, Status string }

type SetStatusIn struct {
	LoadID string `json:"load_id"`
	Status string `json:"status"`
}

// Plan lists every change a write would make, with
// old and new values.
type Plan struct {
	Tool, Target string
	Changes      map[string][2]any
	Notes        []string
}

type Store interface {
	GetLoad(ctx context.Context, id string) (Load, error)
	SetStatus(ctx context.Context, id, status string) error
}

// SetStatus is the live path: it builds the plan
// first, so dry and real runs share the same checks.
// dry comes from config per run, never the model.
func SetStatus(ctx context.Context, db Store,
	in SetStatusIn, dry bool) (string, error) {
	p, err := planSetStatus(ctx, db, in)
	if err != nil {
		return "", err
	}
	old := p.Changes["status"][0]
	summary := fmt.Sprintf("%s %s: status %v -> %v",
		p.Tool, p.Target, old, in.Status)
	if dry {
		return "DRY RUN, nothing changed. " + summary, nil
	}
	// Fail the apply if the old value moved since
	// the plan was made (a stale plan).
	cur, err := db.GetLoad(ctx, p.Target)
	if err != nil {
		return "", err
	}
	if cur.Status != old {
		return "", fmt.Errorf("stale plan: status is %s",
			cur.Status)
	}
	return summary, db.SetStatus(ctx, p.Target, in.Status)
}
