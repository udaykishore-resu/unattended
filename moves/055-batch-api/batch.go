// Package batch reconciles a provider batch against
// what was sent, so missing items are never lost.
//
// Move 55: batch APIs for anything that can wait.
package batch

import (
	"context"

	"github.com/udaykishore-resu/unattended/llm"
)

func Collect(ctx context.Context, b Batcher,
	id string,
	sent map[string]llm.Request) (Outcome, error) {
	res, err := b.Results(ctx, id)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Done: map[string]llm.Response{}}
	for key := range sent {
		it, ok := res[key]
		if !ok || it.Err != nil ||
			it.Resp.StopReason != llm.StopEnd {
			out.Missed = append(out.Missed, key)
			continue
		}
		out.Done[key] = it.Resp
	}
	return out, nil
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

// Outcome splits what you sent into done and missed.
type Outcome struct {
	Done   map[string]llm.Response
	Missed []string
}
