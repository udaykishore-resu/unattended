package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// When both models fail the gate, Classify returns no
// label: the rejected large-model label must not leak
// out as if it were a classification.
func TestHumanPathReturnsNoLabel(t *testing.T) {
	small := &llm.FakeModel{Responses: []llm.Response{resp("billing", "low", "Where is load")}}
	large := &llm.FakeModel{Responses: []llm.Response{resp("billing", "medium", "Where is load")}}
	q := &queue{}
	l, who, err := router(small, large, q).Classify(context.Background(), llm.Request{}, ticket)
	if err != nil || who != "human" || l != (Label{}) || q.n != 1 {
		t.Fatalf("l=%+v who=%s err=%v queued=%d", l, who, err, q.n)
	}
}

type failingQueue struct{}

func (failingQueue) Add(context.Context, llm.Request, string) error {
	return errors.New("review db down")
}

// If filing for review fails, the caller hears it.
func TestReviewAddFailureReturned(t *testing.T) {
	large := &llm.FakeModel{Responses: []llm.Response{resp("billing", "low", "x")}}
	rt := router(&llm.FakeModel{}, large, nil)
	rt.Review = failingQueue{}
	rt.Escalate = func(string) bool { return true }
	l, who, err := rt.Classify(context.Background(), llm.Request{}, ticket)
	if err == nil || who != "human" || l != (Label{}) {
		t.Fatalf("l=%+v who=%s err=%v", l, who, err)
	}
}

// The gate checks that the quote exists, not that the
// label is right: a high, grounded, wrong label is
// accepted. This is why the book samples accepted
// answers and measures accuracy on them.
func TestConfidentlyWrongButGroundedIsAccepted(t *testing.T) {
	damage := "The pallet arrived crushed and two cases are broken."
	small := &llm.FakeModel{Responses: []llm.Response{
		resp("tracking", "high", "The pallet arrived")}} // should be claims
	l, who, _ := router(small, &llm.FakeModel{}, &queue{}).Classify(
		context.Background(), llm.Request{}, damage)
	if who != "small" || l.Queue != "tracking" {
		t.Fatalf("who=%s l=%+v", who, l)
	}
}

// grounded checks this ticket, folding only case and
// spacing; a quote from another ticket, or an
// empty one, fails.
func TestGroundedAgainstTheRightSource(t *testing.T) {
	if !grounded(ticket, "where   IS load hf-1?") {
		t.Fatal("case/space folding")
	}
	for _, ev := range []string{"", "  ", "Customs hold on HF-2", "Where is load HF-1 today"} {
		if grounded(ticket, ev) {
			t.Fatalf("accepted %q", ev)
		}
	}
	rt := router(nil, nil, nil)
	if rt.accept(Label{Queue: "legal", Confidence: "high", Evidence: "Where is load"}, ticket) {
		t.Fatal("queue outside Allowed accepted")
	}
	if rt.accept(Label{Queue: "tracking", Confidence: "High", Evidence: "Where is load"}, ticket) {
		t.Fatal(`confidence must be exactly "high"`)
	}
}
