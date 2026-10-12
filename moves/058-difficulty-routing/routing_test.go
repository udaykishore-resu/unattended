package routing

import (
	"context"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

type queue struct{ n int }

func (q *queue) Add(context.Context, llm.Request, string) error { q.n++; return nil }

func router(small, large *llm.FakeModel, q *queue) Router {
	return Router{Small: small, Large: large, Review: q,
		Allowed: map[string]bool{"tracking": true, "billing": true, "claims": true},
		Escalate: func(src string) bool {
			return strings.Contains(strings.ToLower(src), "customs")
		}}
}

const ticket = "Where is load HF-1? It was due Tuesday."

func resp(queue, conf, ev string) llm.Response {
	return llm.Response{Text: `{"queue":"` + queue + `","confidence":"` +
		conf + `","evidence":"` + ev + `"}`}
}

func TestEasyStaysSmall(t *testing.T) {
	small := &llm.FakeModel{Responses: []llm.Response{
		resp("tracking", "high", "Where is load HF-1?")}}
	large := &llm.FakeModel{}
	l, who, err := router(small, large, &queue{}).Classify(context.Background(), llm.Request{}, ticket)
	if err != nil || who != "small" || l.Queue != "tracking" || len(large.Calls) != 0 {
		t.Fatalf("%+v %s %v large=%d", l, who, err, len(large.Calls))
	}
}

// "high" with a quote that isn't in the ticket is
// not accepted: the model's own number is never the gate.
func TestUngroundedEscalates(t *testing.T) {
	small := &llm.FakeModel{Responses: []llm.Response{
		resp("tracking", "high", "load is delayed by weather")}}
	large := &llm.FakeModel{Responses: []llm.Response{
		resp("tracking", "high", "due Tuesday")}}
	_, who, _ := router(small, large, &queue{}).Classify(context.Background(), llm.Request{}, ticket)
	if who != "large" {
		t.Fatalf("who = %s", who)
	}
}

func TestRulesSkipSmallAndMissesGoToPerson(t *testing.T) {
	small := &llm.FakeModel{}
	large := &llm.FakeModel{Responses: []llm.Response{resp("claims", "medium", "customs")}}
	q := &queue{}
	_, who, err := router(small, large, q).Classify(context.Background(),
		llm.Request{}, "Customs hold on HF-2")
	if err != nil || who != "human" || q.n != 1 || len(small.Calls) != 0 {
		t.Fatalf("who=%s err=%v queued=%d small=%d", who, err, q.n, len(small.Calls))
	}
}

func TestQueueNotAllowed(t *testing.T) {
	small := &llm.FakeModel{Responses: []llm.Response{resp("legal", "high", "Where is load")}}
	large := &llm.FakeModel{Responses: []llm.Response{resp("legal", "high", "Where is load")}}
	q := &queue{}
	_, who, _ := router(small, large, q).Classify(context.Background(), llm.Request{}, ticket)
	if who != "human" || q.n != 1 {
		t.Fatalf("who=%s", who)
	}
}
