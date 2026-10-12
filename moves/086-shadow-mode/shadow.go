// Package shadow runs a candidate model on live
// inputs and keeps its answer only in a log. The
// live path never waits for it.
//
// Move 86: shadow mode before cutover.
package shadow

import (
	"context"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

type Shadow struct {
	Live, Cand llm.Model
	Store      Store
	Sem        chan struct{} // caps shadow work
}

func (s *Shadow) Do(ctx context.Context, id string,
	r llm.Request) (llm.Response, error) {
	live, err := s.Live.Complete(ctx, r)
	if err != nil {
		return live, err
	}
	select {
	case s.Sem <- struct{}{}:
		go s.run(context.WithoutCancel(ctx), id, r, live)
	default: // shed shadow load
	}
	return live, nil
}

func (s *Shadow) run(ctx context.Context, id string,
	r llm.Request, live llm.Response) {
	defer func() { <-s.Sem }() // free the slot
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cand, err := s.Cand.Complete(cctx, r)
	// ctx, not cctx: a timeout still gets its row
	s.Store.Save(ctx, Pair{id, live, cand, err})
}

// Pair holds the ID, both responses and the error.
type Pair struct {
	ID         string
	Live, Cand llm.Response
	Err        error
}

// Store is any sink with that Save method; it logs
// its own write errors.
type Store interface {
	Save(ctx context.Context, p Pair)
}
