// Package enrich attaches the runbook and recent
// deploys to an alert before it pages, under a hard
// timeout, and never blocks the page. The model's
// note gets its own shorter budget, and code drops a
// note that names a cause or a fix.
//
// Move 66: alert enrichment.
package enrich

import (
	"context"
	"regexp"
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
	nctx, stop := context.WithTimeout(ctx,
		3*time.Second)
	defer stop()
	n, err := s.Note(nctx, e)
	if err == nil && !causal.MatchString(n) {
		e.Note = n
	}
	return e
}

// causal matches wording that claims a cause or a
// fix. A note may say "possibly related", never more.
var causal = regexp.MustCompile(
	`(?i)\b(caus(e|ed|es|ing)|root|fix(ed|es)?|due to)\b`)

// Sources is an interface with one method per
// system, the model included, and Alert and Enriched
// are plain structs.

type Sources interface {
	Runbook(ctx context.Context, alert string) (string, error)
	Deploys(ctx context.Context, service string,
		since time.Time) ([]Deploy, error)
	Note(ctx context.Context, e Enriched) (string, error)
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
