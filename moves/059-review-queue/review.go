// Package review is the approval queue: approvals are
// rows with a claim, a decision that commits only for
// the reviewer who holds the claim, and an executor
// that ships only the approved text.
//
// Move 59: human-in-the-loop as a queue, not an
// email.
package review

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Schema is the table the queries assume. seed marks
// a known-bad draft planted to test reviewers.
const Schema = `CREATE TABLE reviews (
  id         bigserial PRIMARY KEY,
  kind       text NOT NULL,
  draft      text NOT NULL,
  evidence   text NOT NULL,
  final      text,
  state      text NOT NULL DEFAULT 'open'
    CHECK (state IN ('open','approved','rejected')),
  seed       boolean NOT NULL DEFAULT false,
  due_by     timestamptz NOT NULL,
  claimed_by text,
  claimed_at timestamptz,
  decided_by text,
  decided_at timestamptz)`

// claimQ hands out the open item due first that no
// one holds, or whose claim is older than 15 minutes
// (Halcyon's default). As printed in the book.
const claimQ = `UPDATE reviews SET claimed_by = $1,
       claimed_at = now()
WHERE id = (
  SELECT id FROM reviews
  WHERE state = 'open'
    AND (claimed_by IS NULL
      OR claimed_at < now() - interval '15 minutes')
  ORDER BY due_by
  LIMIT 1
  FOR UPDATE SKIP LOCKED)
RETURNING id, kind, draft, evidence, due_by`

// decideQ commits a decision only while the item is
// open and this reviewer still holds the claim. A
// reviewer whose claim lapsed and was taken, or who
// submits twice, changes nothing. As printed.
const decideQ = `UPDATE reviews
SET state = $3, final = $4,
    decided_by = $2, decided_at = now()
WHERE id = $1 AND state = 'open'
  AND claimed_by = $2`

// approvedQ is what the executor reads.
const approvedQ = `SELECT final, seed FROM reviews
WHERE id = $1 AND state = 'approved'`

// Item is one claimed review.
type Item struct {
	ID              int64
	Kind, Draft     string
	Evidence        string
	DueBy           time.Time
	Reviewer, Final string
}

// ErrEmpty: nothing to review.
var ErrEmpty = errors.New("review queue empty")

// ErrClaimLost: the item was decided already, or
// another reviewer holds it now. Reload; do not
// retry the same decision.
var ErrClaimLost = errors.New(
	"claim lost: item decided or reclaimed")

// ErrNotApproved: no approved decision for this item.
var ErrNotApproved = errors.New("not approved")

// ErrSeed: a planted known-bad draft. The executor
// refuses it even when a reviewer approved it.
var ErrSeed = errors.New("seed item: never executed")

// Claim takes the next item for reviewer.
func Claim(ctx context.Context, db *sql.DB,
	reviewer string) (Item, error) {
	it := Item{Reviewer: reviewer}
	err := db.QueryRowContext(ctx, claimQ, reviewer).Scan(
		&it.ID, &it.Kind, &it.Draft, &it.Evidence, &it.DueBy)
	if errors.Is(err, sql.ErrNoRows) {
		return it, ErrEmpty
	}
	return it, err
}

// Decide records approve (with the possibly edited
// final text) or reject. It fails with ErrClaimLost
// unless this reviewer still holds an open item.
func Decide(ctx context.Context, db *sql.DB, id int64,
	reviewer, state, final string) error {
	var f any = final
	if state != "approved" {
		f = nil
	}
	res, err := db.ExecContext(ctx, decideQ,
		id, reviewer, state, f)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrClaimLost
	}
	return nil
}

// Execute sends exactly the approved bytes. It never
// calls a model, refuses anything not approved, and
// refuses seeds.
func Execute(ctx context.Context, db *sql.DB, id int64,
	send func(context.Context, string) error) error {
	var final string
	var seed bool
	err := db.QueryRowContext(ctx, approvedQ, id).Scan(
		&final, &seed)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotApproved
	} else if err != nil {
		return err
	}
	if seed {
		return ErrSeed
	}
	return send(ctx, final)
}
