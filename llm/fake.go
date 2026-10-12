package llm

import (
	"context"
	"errors"
	"slices"
	"sync"
)

var ErrNoResponses = errors.New("fake: no responses")

// FakeModel replays canned responses in order and
// records every request with a deep copy of its
// history. Set Responses and Err before the first
// call; Complete reads them under the mutex.
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
	r.Messages = cloneMessages(r.Messages)
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

// cloneMessages copies the history down to each
// tool call's input bytes, so later edits by the
// caller cannot change what the fake recorded.
func cloneMessages(ms []Message) []Message {
	out := slices.Clone(ms)
	for i := range out {
		tcs := slices.Clone(out[i].ToolCalls)
		for j := range tcs {
			tcs[j].Input = slices.Clone(tcs[j].Input)
		}
		out[i].ToolCalls = tcs
	}
	return out
}
