// Package triggers receives push events: check the
// signature, store the event, return 202. Never call
// a model in the handler.
//
// Move 51: event-driven triggers over polling.
package triggers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) mail(w http.ResponseWriter,
	r *http.Request) {
	body, err := io.ReadAll(
		http.MaxBytesReader(w, r.Body, 1<<20))
	sig := r.Header.Get("X-Sig")
	if err != nil || !verify(s.secret, body, sig) ||
		eventID(body) == "" {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	_, err = s.db.ExecContext(r.Context(),
		insertEvent, "mail", eventID(body), body)
	if err != nil {
		http.Error(w, "db",
			http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// Server and the helpers below are described in the
// book's prose; minimal implementations.

type Server struct {
	db     *sql.DB
	secret []byte
}

// Handler exposes the receiver.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.mail)
}

// Schema is the events table the insert assumes. The
// primary key on (source, event_id) is what makes
// ON CONFLICT DO NOTHING drop a redelivery; without
// a unique index Postgres rejects the insert.
const Schema = `CREATE TABLE events (
  source   text NOT NULL,
  event_id text NOT NULL CHECK (event_id <> ''),
  body     bytea NOT NULL,
  received timestamptz NOT NULL DEFAULT now(),
  done     boolean NOT NULL DEFAULT false,
  PRIMARY KEY (source, event_id))`

// insertEvent ends in ON CONFLICT DO NOTHING on
// source and event ID, so redeliveries vanish.
const insertEvent = `INSERT INTO events
  (source, event_id, body) VALUES ($1, $2, $3)
  ON CONFLICT (source, event_id) DO NOTHING`

// verify is an HMAC check with hmac.Equal over the
// decoded hex signature (so case doesn't matter and
// a malformed signature fails).
func verify(secret, body []byte, sig string) bool {
	got, err := hex.DecodeString(sig)
	if err != nil || len(got) != sha256.Size {
		return false
	}
	m := hmac.New(sha256.New, secret)
	m.Write(body)
	return hmac.Equal(m.Sum(nil), got)
}

// eventID reads the provider's message ID. An empty
// result makes the handler refuse the event: an
// empty ID would collapse every such event into one
// row.
func eventID(body []byte) string {
	var e struct {
		MessageID string `json:"message_id"`
	}
	if json.Unmarshal(body, &e) != nil {
		return ""
	}
	return e.MessageID
}

// Msg is one message the mail API lists.
type Msg struct {
	ID   string
	Body []byte
}

// Sweep is the slow poll behind the webhook. It takes
// the message IDs the mail API lists for the last
// window and stores each through the same insert, so
// a message the webhook already stored (or a worker
// already finished) is a no-op. It returns how many
// were new: the number to alert on.
func (s *Server) Sweep(ctx context.Context,
	msgs []Msg) (int, error) {
	n := 0
	for _, m := range msgs {
		if m.ID == "" {
			continue
		}
		res, err := s.db.ExecContext(ctx, insertEvent,
			"mail", m.ID, m.Body)
		if err != nil {
			return n, err
		}
		k, err := res.RowsAffected()
		if err != nil {
			return n, err
		}
		n += int(k)
	}
	return n, nil
}
