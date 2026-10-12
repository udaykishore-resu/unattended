// Package triggers receives push events: check the
// signature, store the event, return 202. Never call
// a model in the handler.
//
// Move 51: event-driven triggers over polling.
package triggers

import (
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
		io.LimitReader(r.Body, 1<<20))
	sig := r.Header.Get("X-Sig")
	if err != nil || !verify(s.secret, body, sig) {
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

// insertEvent ends in ON CONFLICT DO NOTHING on
// source and event ID, so redeliveries vanish.
const insertEvent = `INSERT INTO events
  (source, event_id, body) VALUES ($1, $2, $3)
  ON CONFLICT (source, event_id) DO NOTHING`

// verify is an HMAC check with hmac.Equal.
func verify(secret, body []byte, sig string) bool {
	m := hmac.New(sha256.New, secret)
	m.Write(body)
	want := hex.EncodeToString(m.Sum(nil))
	return hmac.Equal([]byte(want), []byte(sig))
}

// eventID reads the provider's message ID.
func eventID(body []byte) string {
	var e struct {
		MessageID string `json:"message_id"`
	}
	json.Unmarshal(body, &e)
	return e.MessageID
}
