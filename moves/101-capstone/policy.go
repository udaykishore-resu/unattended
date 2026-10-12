package capstone

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

// Route is where one email ends up.
type Route string

const (
	Applied      Route = "applied"       // Green path
	Review       Route = "review"        // L2 queue
	ProviderDown Route = "provider_down" // retry queue
	Off          Route = "off"           // kill switch
)

// Outcome is the result of one run.
type Outcome struct {
	Route   Route
	Why     string
	Updates []Checked
}

// Bot wires the pieces. Model should be a Retrying
// wrapper around the stack from Move 83.
type Bot struct {
	Model   llm.Model
	Enabled func(context.Context) bool // delay_bot_enabled
	Loads   func(id string) (Load, bool)
}

// Process applies the failure policy. A schema or
// business-validation failure gets exactly one
// repair call that carries the validation error. If
// the repair also fails, the email goes to review:
// there is no third model call. Provider errors are
// the retry wrapper's job; once it gives up, the
// email is queued as provider-unavailable. A gate
// rejection goes to review and is never retried.
func (b Bot) Process(ctx context.Context,
	e Email) Outcome {
	if !b.Enabled(ctx) {
		return Outcome{Route: Off, Why: "kill switch"}
	}
	req := BuildRequest(e.Subject, e.Body)
	zone := func(id string) *time.Location {
		l, ok := b.Loads(id)
		if !ok {
			return nil
		}
		return l.Delivery
	}
	ups, err := b.call(ctx, &req, e, zone)
	var verr *ValidationError
	if errors.As(err, &verr) {
		// The one repair: same System, same fence,
		// plus the error.
		req.Messages = append(req.Messages,
			llm.Message{Role: "user", Content: fmt.Sprintf(
				"Your JSON failed validation:\n- %s\n"+
					"Return corrected JSON. If a value "+
					"is not in the email, set status "+
					"to \"not_present\".",
				strings.Join(verr.Problems, "\n- "))})
		ups, err = b.call(ctx, &req, e, zone)
		if errors.As(err, &verr) {
			return Outcome{Route: Review,
				Why: "failed after one repair: " +
					err.Error()}
		}
	}
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return Outcome{Route: ProviderDown,
			Why: err.Error()}
	case err != nil:
		return Outcome{Route: Review, Why: err.Error()}
	}
	for _, c := range ups {
		var l Load
		found := false
		if !c.Abstained() {
			l, found = b.Loads(*c.LoadID)
		}
		if ok, why := Gate(e, c, l, found); !ok {
			return Outcome{Route: Review, Why: why,
				Updates: ups}
		}
	}
	return Outcome{Route: Applied, Updates: ups}
}

// call makes one model call and validates it. The
// assistant turn is appended so a repair sees it.
func (b Bot) call(ctx context.Context,
	req *llm.Request, e Email,
	zone ZoneFor) ([]Checked, error) {
	resp, err := b.Model.Complete(ctx, *req)
	if err != nil {
		return nil, err
	}
	req.Messages = append(req.Messages, llm.Message{
		Role: "assistant", Content: resp.Text})
	if resp.StopReason != llm.StopEnd {
		return nil, &ValidationError{[]string{
			"stop reason " + string(resp.StopReason)}}
	}
	return Validate(resp.Text, e, zone)
}

// StatusError is what an adapter returns for a
// non-2xx HTTP response.
type StatusError struct{ Code int }

func (s StatusError) Error() string {
	return fmt.Sprintf("provider: HTTP %d %s", s.Code,
		http.StatusText(s.Code))
}

var ErrProviderUnavailable = errors.New(
	"provider unavailable")

// Retrying retries transport errors, timeouts, 429
// and 5xx with full-jitter backoff, Attempts times
// in all (3 by default), then wraps the last error
// in ErrProviderUnavailable. Other 4xx responses
// are returned at once: a retry cannot fix them.
type Retrying struct {
	Next     llm.Model
	Attempts int           // 0 means 3
	Base     time.Duration // 0 means 500ms
	Ceil     time.Duration // 0 means 8s
	Timeout  time.Duration // per attempt; 0 means 30s
	// Sleep waits between attempts; tests stub it.
	Sleep func(context.Context, time.Duration) error
}

func (r Retrying) Complete(ctx context.Context,
	req llm.Request) (llm.Response, error) {
	n := orInt(r.Attempts, 3)
	base := orDur(r.Base, 500*time.Millisecond)
	ceil := orDur(r.Ceil, 8*time.Second)
	sleep := r.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	var last error
	for i := range n {
		actx, cancel := context.WithTimeout(ctx,
			orDur(r.Timeout, 30*time.Second))
		resp, err := r.Next.Complete(actx, req)
		cancel()
		if err == nil {
			return resp, nil
		}
		if ctx.Err() != nil {
			return llm.Response{}, ctx.Err()
		}
		if !transient(err) {
			return llm.Response{}, err
		}
		last = err
		if i == n-1 {
			break
		}
		// Cap the shift before it can overflow.
		d := rand.N(min(ceil, base<<min(i, 16)) + 1)
		if err := sleep(ctx, d); err != nil {
			return llm.Response{}, err
		}
	}
	return llm.Response{}, fmt.Errorf("%w after %d "+
		"attempts: %w", ErrProviderUnavailable, n, last)
}

// transient: 429 and 5xx yes, other HTTP statuses
// no, and anything else (a reset connection, a
// per-attempt timeout) yes.
func transient(err error) bool {
	var s StatusError
	if errors.As(err, &s) {
		return s.Code == http.StatusTooManyRequests ||
			s.Code >= 500
	}
	return true
}

func sleepCtx(ctx context.Context,
	d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func orInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func orDur(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}
