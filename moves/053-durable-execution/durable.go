// Package durable journals each step's result, so a
// restarted workflow replays finished steps from the
// journal instead of rerunning them.
//
// This is a journal, not a workflow engine: it has no
// timers, versioning or retries of its own. A step
// that crashes after its side effect but before the
// journal write runs again on replay, so each step
// gets a key (Move 52) to pass downstream.
//
// Move 53: durable execution for long workflows.
package durable

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	idem "github.com/udaykishore-resu/unattended/moves/052-idempotency-keys"
)

func Step[T any](ctx context.Context, r *Run,
	name string,
	fn func(context.Context, string) (T, error),
) (out T, err error) {
	var raw []byte
	err = r.DB.QueryRowContext(ctx, getStep,
		r.ID, name).Scan(&raw)
	if err == nil {
		return out, json.Unmarshal(raw, &out)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	key := idem.Key(name, r.ID) // send it downstream
	if out, err = fn(ctx, key); err == nil {
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

// Schema is the tables the queries assume. The
// primary key on (run_id, name) makes a second
// journal write for one step fail instead of
// overwriting the first.
const Schema = `CREATE TABLE wf_steps (
  run_id text NOT NULL,
  name   text NOT NULL,
  result bytea NOT NULL,
  PRIMARY KEY (run_id, name));
CREATE TABLE wf_runs (
  run_id      text PRIMARY KEY,
  owner       text NOT NULL,
  lease_until timestamptz NOT NULL)`

// getStep and putStep read and write a steps table
// keyed by run ID and step name.
const (
	getStep = `SELECT result FROM wf_steps
  WHERE run_id = $1 AND name = $2`
	putStep = `INSERT INTO wf_steps (run_id, name, result)
  VALUES ($1, $2, $3)`
)

// claimRunQ takes a run for one worker: a new run, an
// expired lease, or a renewal by the same owner.
const claimRunQ = `INSERT INTO wf_runs
  (run_id, owner, lease_until) VALUES ($1, $2, $3)
  ON CONFLICT (run_id) DO UPDATE
  SET owner = $2, lease_until = $3
  WHERE wf_runs.lease_until < now()
     OR wf_runs.owner = $2`

// ErrRunBusy: another worker is replaying this run.
var ErrRunBusy = errors.New("run claimed by another worker")

// Claim takes the run for owner until now+d, so only
// one worker replays a run at a time. Call it before
// replaying, and again to renew. Without it, two
// workers replaying one run can both run a step
// before either journals it.
func (r *Run) Claim(ctx context.Context, owner string,
	d time.Duration) error {
	res, err := r.DB.ExecContext(ctx, claimRunQ,
		r.ID, owner, time.Now().Add(d))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrRunBusy
	}
	return nil
}
