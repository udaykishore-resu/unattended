// Package secrets keeps credentials out of the
// context window: the tool fixes the host and token,
// the model supplies only an allowed relative path,
// and every echoed form of the token is scrubbed.
//
// Move 99: secrets never enter the context window.
package secrets

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Integration struct {
	Base  string   // fixed API root, from config
	Env   string   // env var that holds the token
	Paths []string // allowed relative prefixes
}

func (in Integration) Get(ctx context.Context,
	rel string) (string, error) {
	u, err := url.Parse(rel)
	if err != nil || u.IsAbs() || u.Host != "" ||
		strings.Contains(u.Path, "..") ||
		!hasPrefix(u.Path, in.Paths) {
		return "", errors.New("path not allowed")
	}
	tok := os.Getenv(in.Env)
	if tok == "" {
		return "", errors.New("not configured")
	}
	body, err := fetch(ctx, in.Base+u.Path, tok)
	return scrub(body, tok), err
}

func scrub(s, tok string) string {
	for _, f := range []string{tok,
		base64.StdEncoding.EncodeToString([]byte(tok)),
		base64.URLEncoding.EncodeToString([]byte(tok)),
		url.QueryEscape(tok)} {
		s = strings.ReplaceAll(s, f, "[REDACTED]")
	}
	return s
}

// hasPrefix checks the list.
func hasPrefix(p string, prefixes []string) bool {
	for _, pre := range prefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// fetch is the HTTP helper that sets the bearer
// header. It returns the body even on a non-2xx
// status, because error bodies echo tokens too.
func fetch(ctx context.Context, u,
	tok string) (string, error) {
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err == nil && resp.StatusCode >= 300 {
		err = errors.New(resp.Status)
	}
	return string(b), err
}
