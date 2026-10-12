package emailrecord

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrNoMessageID sends an email to review: without a
// usable Message-Id there is no safe idempotency key.
var ErrNoMessageID = errors.New("missing or malformed Message-Id")

// Key is the idempotency key for the i-th request
// (0-based) in one email. Its scope is the mailbox,
// the Message-Id and the position, so a redelivered
// email gets the same keys and two tenders in one
// email get different ones. The key marks the email's
// request as applied, whether that created a record or
// updated an earlier one (change_to_existing).
func Key(mailbox, msgID string, i int) (string, error) {
	id := strings.TrimSpace(msgID)
	if !validMsgID(id) || mailbox == "" || i < 0 {
		return "", ErrNoMessageID
	}
	return fmt.Sprintf("%s|%s|%d", mailbox, id, i), nil
}

// validMsgID accepts <left@right> with no spaces and
// one @, which is what RFC 5322 msg-id looks like in
// practice. Anything else fails closed.
func validMsgID(id string) bool {
	if len(id) < 5 || len(id) > 998 ||
		id[0] != '<' || id[len(id)-1] != '>' {
		return false
	}
	in := id[1 : len(id)-1]
	at := strings.IndexByte(in, '@')
	return at > 0 && at < len(in)-1 &&
		strings.Count(in, "@") == 1 &&
		!strings.ContainsAny(in, " \t\r\n<>")
}

// State is where every email ends.
type State string

const (
	Created     State = "created"
	Updated     State = "updated"
	NeedsReview State = "needs_review"
	Ignored     State = "ignored"
)

// Intake records the final state per key. The real
// table has a unique index on key; this map stands
// in for it in tests.
type Intake struct {
	mu   sync.Mutex
	done map[string]State
}

// Apply runs write once per key. A retried or
// redelivered email finds its key and gets the saved
// state back without writing again.
func (s *Intake) Apply(key string,
	write func() (State, error)) (State, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.done[key]; ok {
		return st, true, nil
	}
	st, err := write()
	if err != nil {
		return "", false, err // not recorded: retry later
	}
	if s.done == nil {
		s.done = map[string]State{}
	}
	s.done[key] = st
	return st, false, nil
}
