package triggers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

func newSrv(t *testing.T) (*events, *httptest.Server) {
	ev := &events{rows: map[string]string{}}
	srv := httptest.NewServer((&Server{db: sqlfake.Open(ev.h), secret: secret}).Handler())
	t.Cleanup(srv.Close)
	return ev, srv
}

// A body over 1 MiB is refused, even when signed, and
// nothing is stored.
func TestOversizedBodyRefused(t *testing.T) {
	ev, srv := newSrv(t)
	big := `{"message_id":"m-big","pad":"` + strings.Repeat("x", 1<<20) + `"}`
	if code := post(t, srv, big, sign(big)); code != http.StatusBadRequest {
		t.Fatalf("code = %d", code)
	}
	if len(ev.rows) != 0 {
		t.Fatal("oversized event stored")
	}
}

func TestMalformedSignatures(t *testing.T) {
	ev, srv := newSrv(t)
	body := `{"message_id":"m-3"}`
	good := sign(body)
	for _, sig := range []string{
		"", "zz", "not-hex-at-all", good[:10], good + "00",
		sign(body + " "), // signed over a different body
	} {
		if code := post(t, srv, body, sig); code != http.StatusBadRequest {
			t.Fatalf("sig %q: code = %d", sig, code)
		}
	}
	// Upper-case hex of the right MAC is the same bytes.
	if code := post(t, srv, body, strings.ToUpper(good)); code != http.StatusAccepted {
		t.Fatalf("upper-case hex: code = %d", code)
	}
	if len(ev.rows) != 1 {
		t.Fatalf("rows = %d", len(ev.rows))
	}
}

// A signed event with no ID (or non-JSON) is refused:
// stored under "" it would swallow every later
// ID-less event as a duplicate.
func TestMissingEventIDRefused(t *testing.T) {
	ev, srv := newSrv(t)
	for _, body := range []string{`{"subject":"truck late"}`, `{"message_id":""}`, `not json`} {
		if code := post(t, srv, body, sign(body)); code != http.StatusBadRequest {
			t.Fatalf("%s: code = %d", body, code)
		}
	}
	if len(ev.rows) != 0 {
		t.Fatalf("rows = %d", len(ev.rows))
	}
}

// Concurrent redeliveries of one event store one row.
func TestConcurrentRedeliveries(t *testing.T) {
	ev, srv := newSrv(t)
	body := `{"message_id":"<m-9@carrier>"}`
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if code := post(t, srv, body, sign(body)); code != http.StatusAccepted {
				t.Errorf("code = %d", code)
			}
		})
	}
	wg.Wait()
	if len(ev.rows) != 1 {
		t.Fatalf("rows = %d", len(ev.rows))
	}
}

// The sweeper finds what the webhook dropped and
// re-inserts nothing the webhook already stored.
func TestSweepFindsOnlyMissed(t *testing.T) {
	ev, srv := newSrv(t)
	var listed []Msg
	for i := range 100 {
		id := fmt.Sprintf("m-%03d", i)
		body := `{"message_id":"` + id + `"}`
		listed = append(listed, Msg{ID: id, Body: []byte(body)})
		if i%5 != 0 { // the provider dropped every fifth push
			post(t, srv, body, sign(body))
		}
	}
	s := &Server{db: sqlfake.Open(ev.h), secret: secret}
	n, err := s.Sweep(context.Background(), listed)
	if err != nil || n != 20 || len(ev.rows) != 100 {
		t.Fatalf("new=%d err=%v rows=%d", n, err, len(ev.rows))
	}
	// A second sweep over the same window finds nothing.
	if n, _ := s.Sweep(context.Background(), listed); n != 0 {
		t.Fatalf("second sweep new=%d", n)
	}
}
