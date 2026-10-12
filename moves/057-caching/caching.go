// Package caching builds a response-cache key that
// covers everything that changes the answer.
//
// Move 57: response caching and prompt caching for
// shared prefixes.
package caching

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

func CacheKey(tenant, model, ver string,
	r llm.Request) (string, error) {
	b, err := json.Marshal(struct {
		Tenant, Model, Ver string
		Req                llm.Request
	}{tenant, model, ver, r})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// SortTools keeps the cacheable prefix stable: tools
// built from a Go map come out in random order.
// Named in the "Breaks when" list; minimal.
func SortTools(ts []llm.Tool) {
	slices.SortFunc(ts, func(a, b llm.Tool) int {
		return strings.Compare(a.Name, b.Name)
	})
}
