// Package idem runs a side-effecting step once per
// key, however often it is retried or redelivered.
//
// Move 52: idempotency keys on every AI step.
package idem

import (
	"context"
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
	fn func(context.Context) ([]byte, error),
) ([]byte, error) {
	ok, err := claim(ctx, db, key)
	if err != nil {
		return nil, err
	} else if !ok {
		return saved(ctx, db, key) // or ErrInFlight
	}
	out, err := fn(ctx)
	if err != nil {
		db.ExecContext(ctx, releaseQ, key)
		return nil, err
	}
	_, err = db.ExecContext(ctx, finishQ, key, out)
	return out, err
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

// Lease must sit well above the step's p99.
var Lease = 5 * time.Minute

// The four queries, for Postgres. claimQ inserts the
// key, or takes over an expired lease on an
// unfinished row; it affects one row only if this
// worker now owns the key.
const (
	claimQ = `INSERT INTO ai_steps (key, lease_until)
  VALUES ($1, $2)
  ON CONFLICT (key) DO UPDATE SET lease_until = $2
  WHERE ai_steps.output IS NULL
    AND ai_steps.lease_until < now()`
	savedQ   = `SELECT output FROM ai_steps WHERE key = $1`
	releaseQ = `UPDATE ai_steps SET lease_until = now()
  WHERE key = $1 AND output IS NULL`
	finishQ = `UPDATE ai_steps SET output = $2
  WHERE key = $1`
)

func claim(ctx context.Context, db *sql.DB,
	key string) (bool, error) {
	res, err := db.ExecContext(ctx, claimQ, key,
		time.Now().Add(Lease))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
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
