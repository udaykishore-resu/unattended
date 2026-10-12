// Package audit writes a tamper-evident audit log:
// each entry stores the hash of the one before it,
// chained per segment (retention class + day + pod),
// so retention can drop a whole segment and the rest
// still verify. Signed checkpoints, held where the
// writer cannot reach, make a rewritten chain show.
//
// Move 97: audit-log every agent action.
package audit

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// sum hashes e with Hash empty. Write and Verify
// both call it, so both hash the same bytes.
func (e Entry) sum() (string, error) {
	e.Hash = ""
	b, err := json.Marshal(e)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), err
}

// Write chains e into its segment. mu guards last,
// the map from segment to its newest hash.
func (l *Log) Write(e Entry) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e.At = time.Now().UTC()
	day := e.At.Format(time.DateOnly)
	e.Seg = e.Class + "/" + day + "/" + l.pod
	e.Prev = l.last[e.Seg]
	if e.Hash, err = e.sum(); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err = l.w.Write(append(b, '\n')); err != nil {
		return err
	}
	l.last[e.Seg] = e.Hash
	return nil
}

// Log is described in the book's prose: a writer w,
// the pod name, and last, guarded by mu.
type Log struct {
	mu   sync.Mutex
	w    io.Writer
	pod  string
	last map[string]string
}

// NewLog starts a writer for one pod. tails holds
// each open segment's newest hash, loaded from the
// table on start; nil for a fresh log. Without it, a
// restarted pod would start a second chain.
func NewLog(w io.Writer, pod string,
	tails map[string]string) *Log {
	last := map[string]string{}
	for seg, h := range tails {
		last[seg] = h
	}
	return &Log{w: w, pod: pod, last: last}
}

// Tails reads a log and returns each segment's newest
// hash: what a restarting pod loads into NewLog.
func Tails(r io.Reader) (map[string]string, error) {
	last := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		last[e.Seg] = e.Hash
	}
	return last, sc.Err()
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

// Do writes a pending entry, runs act only if that
// write succeeded, then writes the outcome. A failed
// audit write blocks the action.
func (l *Log) Do(e Entry, act func() error) error {
	e.Result = "pending"
	if err := l.Write(e); err != nil {
		return fmt.Errorf("audit write failed, "+
			"action not run: %w", err)
	}
	err := act()
	e.Result = "ok"
	if err != nil {
		e.Result = "error: " + err.Error()
	}
	if werr := l.Write(e); werr != nil {
		return errors.Join(err, werr)
	}
	return err
}

// Verify re-reads the log, recomputes every hash with
// sum and checks each segment's chain against its
// checkpoint (segment -> last hash). Feed it the rows
// of closed segments and their checkpoints. A segment
// with rows but no checkpoint, or a checkpoint whose
// rows are gone, is reported.
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
		got, err := e.sum()
		if err != nil || got != e.Hash {
			return fmt.Errorf("line %d: hash mismatch", n)
		}
		last[e.Seg] = e.Hash
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
	for seg := range checkpoints {
		if _, ok := last[seg]; !ok {
			return fmt.Errorf("segment %s: rows missing", seg)
		}
	}
	return nil
}

// Checkpoint is a closed segment's last hash, signed
// at day close by a key the writer cannot reach.
type Checkpoint struct {
	Seg, Hash string
	Sig       []byte
}

// Sign is the day-close job.
func Sign(key ed25519.PrivateKey, seg,
	hash string) Checkpoint {
	msg := []byte(seg + "\n" + hash)
	return Checkpoint{Seg: seg, Hash: hash,
		Sig: ed25519.Sign(key, msg)}
}

// Trusted checks every signature and returns the map
// Verify takes. One bad signature fails them all.
func Trusted(pub ed25519.PublicKey,
	cps []Checkpoint) (map[string]string, error) {
	out := map[string]string{}
	for _, c := range cps {
		msg := []byte(c.Seg + "\n" + c.Hash)
		if !ed25519.Verify(pub, msg, c.Sig) {
			return nil, fmt.Errorf("segment %s: bad "+
				"checkpoint signature", c.Seg)
		}
		out[c.Seg] = c.Hash
	}
	return out, nil
}
