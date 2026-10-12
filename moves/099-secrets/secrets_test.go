package secrets

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

const tok = "sk_live_4dQ+x/9m=="

// portal echoes the token back in several encodings,
// the way careless error bodies do.
func portal(t *testing.T, hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+tok {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request for token " + tok +
			" b64=" + base64.StdEncoding.EncodeToString([]byte(tok)) +
			" url64=" + base64.URLEncoding.EncodeToString([]byte(tok)) +
			" q=" + url.QueryEscape(tok) + " path=" + r.URL.Path))
	}))
}

func TestRefusesAbsoluteAndOffListPaths(t *testing.T) {
	var hits atomic.Int32
	srv := portal(t, &hits)
	defer srv.Close()
	t.Setenv("PORTAL_TOKEN", tok)
	in := Integration{Base: srv.URL, Env: "PORTAL_TOKEN", Paths: []string{"/v1/status/"}}
	for _, rel := range []string{
		"https://attacker.test/v1/status/1", // absolute URL
		"//attacker.test/v1/status/1",       // scheme-relative host
		"/v1/status/../../admin",            // dot-dot
		"/v1/admin/keys",                    // off the list
		srv.URL + "/v1/status/1",            // even our own host, absolute
	} {
		if _, err := in.Get(context.Background(), rel); err == nil {
			t.Errorf("%q was allowed", rel)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("refused paths still reached the network %d times", hits.Load())
	}
}

func TestScrubsEncodedSecrets(t *testing.T) {
	var hits atomic.Int32
	srv := portal(t, &hits)
	defer srv.Close()
	t.Setenv("PORTAL_TOKEN", tok)
	in := Integration{Base: srv.URL, Env: "PORTAL_TOKEN", Paths: []string{"/v1/status/"}}
	body, err := in.Get(context.Background(), "/v1/status/HF-1")
	if err == nil {
		t.Fatal("want the 400 as an error")
	}
	for _, form := range []string{tok, base64.StdEncoding.EncodeToString([]byte(tok)),
		base64.URLEncoding.EncodeToString([]byte(tok)), url.QueryEscape(tok)} {
		if strings.Contains(body, form) {
			t.Fatalf("token form %q reached the model:\n%s", form, body)
		}
	}
	if strings.Count(body, "[REDACTED]") != 4 || !strings.Contains(body, "path=/v1/status/HF-1") {
		t.Fatalf("body = %s", body)
	}
}

func TestNotConfigured(t *testing.T) {
	t.Setenv("PORTAL_TOKEN", "")
	in := Integration{Base: "http://127.0.0.1:1", Env: "PORTAL_TOKEN", Paths: []string{"/"}}
	if _, err := in.Get(context.Background(), "/x"); err == nil ||
		err.Error() != "not configured" {
		t.Fatalf("got %v", err)
	}
}
