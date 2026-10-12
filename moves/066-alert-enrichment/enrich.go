// Package enrich attaches the runbook and recent
// deploys to an alert before it pages, under a hard
// timeout, and never blocks the page.
//
// Move 66: alert enrichment.
package enrich

import (
	"context"
	"time"
)

func Enrich(ctx context.Context, s Sources,
	a Alert) Enriched {
	ctx, cancel := context.WithTimeout(ctx,
		8*time.Second)
	defer cancel()
	e := Enriched{Alert: a}
	rb, err := s.Runbook(ctx, a.Name)
	if err != nil {
		e.Missing = append(e.Missing, "runbook")
	}
	e.Runbook = rb
	since := a.FiredAt.Add(-2 * time.Hour)
	ds, err := s.Deploys(ctx, a.Service, since)
	if err != nil {
		e.Missing = append(e.Missing, "deploys")
	}
	e.Deploys = ds
	return e
}

// Sources is an interface with one method per
// system, and Alert and Enriched are plain structs.

type Sources interface {
	Runbook(ctx context.Context, alert string) (string, error)
	Deploys(ctx context.Context, service string,
		since time.Time) ([]Deploy, error)
}

type Alert struct {
	Name, Service string
	FiredAt       time.Time
}

type Deploy struct {
	Service, Ref string
	At           time.Time
}

type Enriched struct {
	Alert   Alert
	Runbook string
	Deploys []Deploy
	Missing []string
	Note    string // the model's note; may stay empty
}
