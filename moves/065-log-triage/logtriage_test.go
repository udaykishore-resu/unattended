package logtriage

import (
	"fmt"
	"testing"
)

func TestTemplate(t *testing.T) {
	cases := map[string]string{
		"quote LD-88213 failed after 412ms":                                              "quote <id> failed after <n>ms",
		`user "ops@halcyon.test" from 10.2.3.4 req 7f1e2d3c-1111-2222-3333-444455556666`: `user "<s>" from <ip> req <uuid>`,
		"retry 3 of 5 in 1.5s":                                                           "retry <n> of <n> in <n>s",
	}
	for in, want := range cases {
		if got := Template(in); got != want {
			t.Errorf("Template(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

// 400,000 harmless retry warnings bury 900 "pool
// exhausted" lines. Counting finds the new cluster.
func TestNewClusterStandsOut(t *testing.T) {
	var base, now []string
	for i := range 4000 {
		w := fmt.Sprintf("retry carrier %d after %dms", i%40, i%300)
		base = append(base, w)
		now = append(now, w)
	}
	for i := range 9 {
		now = append(now, fmt.Sprintf("quote pool exhausted: %d waiting", 30+i))
	}
	got := New(Clusters(now), Clusters(base))
	if len(got) != 1 || got[0].Template != "quote pool exhausted: <n> waiting" ||
		got[0].Count != 9 || len(got[0].Samples) != 3 {
		t.Fatalf("new clusters = %+v", got)
	}
}
