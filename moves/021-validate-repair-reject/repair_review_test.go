package repair

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// thenErr answers once, then fails every later call
// with err, like a provider timing out on the repair.
type thenErr struct {
	first llm.Response
	err   error
	n     int
}

func (m *thenErr) Complete(context.Context, llm.Request) (llm.Response, error) {
	m.n++
	if m.n == 1 {
		return m.first, nil
	}
	return llm.Response{}, m.err
}

// Breaks when (Move 21): a timed-out repair must not be
// filed as a reject. Extract returns the transport
// error unchanged, for the retry wrapper (Move 56).
func TestTimedOutRepairIsNotAReject(t *testing.T) {
	bad := strings.Replace(good, `"45402"`, `"4540"`, 1)
	m := &thenErr{first: call("c1", bad), err: context.DeadlineExceeded}
	_, err := Extract(context.Background(), m, llm.Request{})
	if errors.Is(err, ErrRejected) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if m.n != 2 {
		t.Fatalf("calls = %d, want 2 (no third)", m.n)
	}
}
