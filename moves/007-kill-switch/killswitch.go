// Package killswitch is a per-action, fail-closed
// flag read from Postgres.
//
// Move 7: design the kill switch before the
// feature.
package killswitch

import (
	"context"
	"database/sql"
	"time"
)

type Gate struct {
	DB   *sql.DB
	Name string // row in automation_flags
}

// Enabled fails closed: no row, no answer, or a
// slow database all mean "off".
func (g Gate) Enabled(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx,
		500*time.Millisecond)
	defer cancel()
	var on bool
	err := g.DB.QueryRowContext(ctx,
		`SELECT enabled FROM automation_flags
         WHERE name = $1`, g.Name).Scan(&on)
	return err == nil && on
}

func Handle(ctx context.Context, g Gate, q Queue,
	it Item) error {
	if !g.Enabled(ctx) {
		return q.ToHuman(ctx, it, "automation off")
	}
	return act(ctx, it)
}

// The book calls Queue and act stubs: "your work
// queue and the bot's real step". Minimal versions:

// Item is one unit of work.
type Item struct{ ID string }

// Queue sends work back to people.
type Queue interface {
	ToHuman(ctx context.Context, it Item,
		reason string) error
}

// act is the bot's real step. A variable so tests
// can count calls.
var act = func(ctx context.Context, it Item) error {
	return nil
}
