package llm

import (
	"context"
	"errors"
	"slices"
	"sync"
)

var ErrNoResponses = errors.New("fake: no responses")

// FakeModel replays canned responses in order and
// records a copy of every request it sees.
type FakeModel struct {
	mu        sync.Mutex
	Responses []Response
	Err       error
	Calls     []Request
}

func (f *FakeModel) Complete(
	ctx context.Context, r Request,
) (Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Copy: an agent loop appends to Messages.
	r.Messages = slices.Clone(r.Messages)
	f.Calls = append(f.Calls, r)
	if f.Err != nil {
		return Response{}, f.Err
	}
	if len(f.Responses) == 0 {
		return Response{}, ErrNoResponses
	}
	resp := f.Responses[0]
	f.Responses = f.Responses[1:]
	if resp.StopReason == "" {
		resp.StopReason = StopEnd
		if len(resp.ToolCalls) > 0 {
			resp.StopReason = StopToolUse
		}
	}
	return resp, nil
}
