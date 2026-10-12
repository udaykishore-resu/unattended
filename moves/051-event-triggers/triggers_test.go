package triggers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

var secret = []byte("s3cret")

func sign(body string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(body))
	return hex.EncodeToString(m.Sum(nil))
}

// events emulates the unique index on (source, event_id).
type events struct {
	mu   sync.Mutex
	rows map[string]string
	down bool
}

func (e *events) h(q string, args []any) ([]string, [][]any, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.down {
		return nil, nil, errors.New("db down")
	}
	k := args[0].(string) + "/" + args[1].(string)
	if _, dup := e.rows[k]; dup {
		return nil, nil, nil // ON CONFLICT DO NOTHING
	}
	e.rows[k] = string(args[2].([]byte))
	return nil, [][]any{{1}}, nil
}

func post(t *testing.T, srv *httptest.Server, body, sig string) int {
	req, _ := http.NewRequest("POST", srv.URL, strings.NewReader(body))
	req.Header.Set("X-Sig", sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestStoreAndAccept(t *testing.T) {
	ev := &events{rows: map[string]string{}}
	s := &Server{db: sqlfake.Open(ev.h), secret: secret}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	body := `{"message_id":"<m-1@carrier>","subject":"truck late"}`
	// A provider redelivers: one stored row.
	for range 3 {
		if code := post(t, srv, body, sign(body)); code != http.StatusAccepted {
			t.Fatalf("code = %d", code)
		}
	}
	if len(ev.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(ev.rows))
	}
}

func TestBadSignature(t *testing.T) {
	ev := &events{rows: map[string]string{}}
	srv := httptest.NewServer((&Server{db: sqlfake.Open(ev.h), secret: secret}).Handler())
	defer srv.Close()
	if code := post(t, srv, `{"message_id":"x"}`, "deadbeef"); code != http.StatusBadRequest {
		t.Fatalf("code = %d", code)
	}
	if len(ev.rows) != 0 {
		t.Fatal("unsigned event stored")
	}
}

// If the row can't be stored, say so; the provider retries.
func TestDatabaseDown(t *testing.T) {
	ev := &events{rows: map[string]string{}, down: true}
	srv := httptest.NewServer((&Server{db: sqlfake.Open(ev.h), secret: secret}).Handler())
	defer srv.Close()
	body := `{"message_id":"m-2"}`
	if code := post(t, srv, body, sign(body)); code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d", code)
	}
}
