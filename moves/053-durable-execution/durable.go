// Package durable journals each step's result, so a
// restarted workflow replays finished steps from the
// journal instead of rerunning them.
//
// Move 53: durable execution for long workflows.
package durable

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func Step[T any](ctx context.Context, r *Run,
	name string,
	fn func(context.Context) (T, error),
) (out T, err error) {
	var raw []byte
	err = r.DB.QueryRowContext(ctx, getStep,
		r.ID, name).Scan(&raw)
	if err == nil {
		return out, json.Unmarshal(raw, &out)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	if out, err = fn(ctx); err == nil {
		raw, err = json.Marshal(out)
	}
	if err != nil {
		return out, err
	}
	_, err = r.DB.ExecContext(ctx, putStep,
		r.ID, name, raw)
	return out, err
}

// Run holds the run ID and a *sql.DB.
type Run struct {
	ID string
	DB *sql.DB
}

// getStep and putStep read and write a steps table
// keyed by run ID and step name.
const (
	getStep = `SELECT result FROM wf_steps
  WHERE run_id = $1 AND name = $2`
	putStep = `INSERT INTO wf_steps (run_id, name, result)
  VALUES ($1, $2, $3)`
)
