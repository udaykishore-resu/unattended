// Package batch reconciles a provider batch against
// what was sent, so a missing item shows up as
// missed instead of vanishing.
//
// Move 55: batch APIs for anything that can wait.
package batch

import (
	"context"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

func Collect(ctx context.Context, b Batcher,
	id string, want llm.StopReason,
	sent map[string]llm.Request) (Outcome, error) {
	res, err := b.Results(ctx, id)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Done: map[string]llm.Response{}}
	for key := range sent {
		it, ok := res[key]
		if !ok || it.Err != nil ||
			it.Resp.StopReason != want {
			out.Missed = append(out.Missed, key)
			continue
		}
		out.Done[key] = it.Resp
	}
	return out, nil
}

// ToSend builds the upload and refuses a duplicate
// custom ID: a map would keep only the last request
// and one item would vanish before Collect could see
// it.
func ToSend(ps []Pending) (map[string]llm.Request,
	error) {
	sent := map[string]llm.Request{}
	for _, p := range ps {
		if _, dup := sent[p.Key]; dup {
			return nil, fmt.Errorf("dup id %s", p.Key)
		}
		sent[p.Key] = p.Req
	}
	return sent, nil
}

// Pending is one request for the batch, keyed by its
// Move 52 idempotency key.
type Pending struct {
	Key string
	Req llm.Request
}

// Batcher wraps the provider's SDK: Submit, Status
// and Results. Results returns one Item per custom
// ID. Described in prose; minimal.
type Batcher interface {
	Submit(ctx context.Context,
		reqs map[string]llm.Request) (string, error)
	Status(ctx context.Context, id string) (string, error)
	Results(ctx context.Context,
		id string) (map[string]Item, error)
}

// Item is the llm.Response, or the provider's
// per-item error mapped to a Go error.
type Item struct {
	Resp llm.Response
	Err  error
}

// Outcome splits what you sent into done and missed:
// every sent ID lands in exactly one of the two.
// Missed covers no result, a per-item error, and any
// stop reason other than want (truncated, refused).
type Outcome struct {
	Done   map[string]llm.Response
	Missed []string
}
