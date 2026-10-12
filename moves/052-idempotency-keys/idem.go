// Package idem runs a side-effecting step once per
// key, however often it is retried or redelivered,
// as far as this database can see. The key is also
// handed to the step, because only a downstream
// system that honors it can stop a repeat that
// happens after a lease expires.
//
// Move 52: idempotency keys on every AI step.
package idem

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

func Once(ctx context.Context, db *sql.DB,
	key string,
	fn func(context.Context, string) ([]byte, error),
) ([]byte, error) {
	l, err := claim(ctx, db, key)
	if err != nil {
		return nil, err
	} else if l == nil {
		return saved(ctx, db, key) // or ErrInFlight
	}
	out, err := fn(ctx, key) // key goes downstream
	// Record even if ctx ended during fn.
	bg := context.WithoutCancel(ctx)
	if err != nil {
		return nil, errors.Join(err, l.release(bg))
	}
	err = db.QueryRowContext(bg, finishQ,
		key, out).Scan(&out)
	if err != nil {
		// Done but not recorded: keep the lease.
		return out, errors.Join(ErrUnrecorded, err)
	}
	return out, nil
}

// Key builds a key from the business identity of
// the work: the step name plus what it acts on.
// Each part is length-prefixed, so "ab"+"c" and
// "a"+"bc" differ. Time, attempt numbers and the
// prompt version stay out. Described in prose;
// minimal implementation.
func Key(step string, parts ...string) string {
	var b strings.Builder
	for _, p := range append([]string{step}, parts...) {
		fmt.Fprintf(&b, "%d:%s|", len(p), p)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}

// ErrInFlight: another worker holds a live lease.
var ErrInFlight = errors.New("step in flight")

// ErrUnrecorded: the step ran (its side effect may
// have happened) but storing its output failed. The
// lease is kept, so nobody reruns the step until it
// expires; the takeover then calls fn again with the
// same key, and only a downstream system that
// honors the key turns that into a no-op. Alert on
// it; do not retry it in a loop.
var ErrUnrecorded = errors.New(
	"step done but not recorded")

// Lease must sit well above the step's p99.
var Lease = 5 * time.Minute

// Schema is the table the queries assume.
const Schema = `CREATE TABLE ai_steps (
  key         text PRIMARY KEY,
  lease_until timestamptz NOT NULL,
  owner       text NOT NULL,
  output      bytea)`

// The four queries, for Postgres. claimQ inserts the
// key, or takes over an expired lease on an
// unfinished row; it affects one row only if this
// caller now owns the key. owner is a random token
// per call, so a stale holder can't release a lease
// a takeover now holds. finishQ keeps the first
// stored output (a stale holder and a takeover may
// both finish) and returns it.
const (
	claimQ = `INSERT INTO ai_steps (key, lease_until, owner)
  VALUES ($1, $2, $3)
  ON CONFLICT (key) DO UPDATE
  SET lease_until = $2, owner = $3
  WHERE ai_steps.output IS NULL
    AND ai_steps.lease_until < now()`
	savedQ   = `SELECT output FROM ai_steps WHERE key = $1`
	releaseQ = `UPDATE ai_steps SET lease_until = now()
  WHERE key = $1 AND owner = $2 AND output IS NULL`
	finishQ = `UPDATE ai_steps
  SET output = COALESCE(output, $2)
  WHERE key = $1 RETURNING output`
)

// lease is a claim this call holds.
type lease struct {
	db       *sql.DB
	key, tok string
}

func claim(ctx context.Context, db *sql.DB,
	key string) (*lease, error) {
	tok := rand.Text()
	res, err := db.ExecContext(ctx, claimQ, key,
		time.Now().Add(Lease), tok)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return nil, err
	}
	return &lease{db, key, tok}, nil
}

// release drops the lease so a retry can run at
// once, but only if this call still owns it.
func (l *lease) release(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx, releaseQ,
		l.key, l.tok)
	return err
}

func saved(ctx context.Context, db *sql.DB,
	key string) ([]byte, error) {
	var out []byte
	err := db.QueryRowContext(ctx, savedQ, key).Scan(&out)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrInFlight
	}
	return out, nil
}
