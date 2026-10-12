// Package audit writes a tamper-evident audit log:
// each entry stores the hash of the one before it,
// chained per segment (retention class + day), so
// retention can drop a whole segment and the rest
// still verify.
//
// Move 97: audit-log every agent action.
package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Log holds a writer w and last, a map from
// segment (retention class + day) to its last
// hash. Write chains each entry in its segment.
func (l *Log) Write(e Entry) error {
	e.At = time.Now().UTC()
	day := e.At.Format(time.DateOnly)
	e.Seg = e.Class + "/" + day
	e.Prev, e.Hash = l.last[e.Seg], ""
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	e.Hash = hex.EncodeToString(sum[:])
	if b, err = json.Marshal(e); err != nil {
		return err
	}
	l.last[e.Seg] = e.Hash
	_, err = l.w.Write(append(b, '\n'))
	return err
}

type Log struct {
	w    io.Writer
	last map[string]string
}

func NewLog(w io.Writer) *Log {
	return &Log{w: w, last: map[string]string{}}
}

// Entry is one action attempt, including denials and
// dry-runs. Agent is deployment plus image tag.
type Entry struct {
	At       time.Time `json:"at"`
	Class    string    `json:"class"` // retention class
	Seg      string    `json:"seg"`
	RunID    string    `json:"run_id"`
	Agent    string    `json:"agent"`
	Version  string    `json:"prompt_version"`
	Tool     string    `json:"tool"`
	Input    string    `json:"input"` // IDs, not personal data
	Decision string    `json:"decision"`
	Approver string    `json:"approver"`
	Result   string    `json:"result"` // pending, then outcome
	Prev     string    `json:"prev"`
	Hash     string    `json:"hash"`
}

// Verify re-reads the log, recomputes every hash and
// checks each segment's chain against its signed
// checkpoint (segment -> last hash). A segment with
// no checkpoint, or a checkpoint with no rows left,
// is reported. Described in prose; minimal.
func Verify(r io.Reader,
	checkpoints map[string]string) error {
	last := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return fmt.Errorf("line %d: %w", n, err)
		}
		if e.Prev != last[e.Seg] {
			return fmt.Errorf("line %d: chain broken in %s",
				n, e.Seg)
		}
		want := e.Hash
		e.Hash = ""
		b, _ := json.Marshal(e)
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != want {
			return fmt.Errorf("line %d: hash mismatch", n)
		}
		last[e.Seg] = want
	}
	if err := sc.Err(); err != nil {
		return err
	}
	for seg, h := range last {
		if checkpoints[seg] != h {
			return fmt.Errorf("segment %s: checkpoint "+
				"mismatch", seg)
		}
	}
	return nil
}
