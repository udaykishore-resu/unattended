package prompts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestVersionIsContentHash(t *testing.T) {
	p, err := Load("classify")
	if err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile("classify.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(disk)
	if want := hex.EncodeToString(sum[:]); p.Version != want {
		t.Fatalf("version %s, want %s", p.Version, want)
	}
	// The full digest: 32 bytes, 64 hex characters, no truncation.
	if len(p.Version) != 64 || p.Body != string(disk) {
		t.Fatalf("prompt = %+v", p)
	}
	// Same bytes, same version: a hash cannot be forgotten like a number.
	again, err := Load("classify")
	if err != nil || again.Version != p.Version {
		t.Fatal("version not stable")
	}
	// One changed byte, a different version.
	edited := append([]byte(p.Body), '.')
	if Hash(edited) == p.Version {
		t.Fatal("an edit kept the same version")
	}
}

func TestMissingPrompt(t *testing.T) {
	if _, err := Load("nope"); err == nil {
		t.Fatal("want an error for an unknown prompt")
	}
}

// The rendered hash is taken over the exact strings
// handed to the client, after every substitution: the
// bytes the model sees. Two renders of one template
// share a file hash but not a rendered hash.
func TestRenderedHashCoversSentBytes(t *testing.T) {
	p, err := Load("classify")
	if err != nil {
		t.Fatal(err)
	}
	f := &llm.FakeModel{Responses: []llm.Response{{Text: "a"}, {Text: "b"}}}
	var logged []string
	for _, email := range []string{"where is HF-1", "where is HF-2"} {
		req := llm.Request{
			System:   p.Body,
			Messages: []llm.Message{{Role: "user", Content: "<email>\n" + email + "\n</email>"}},
		}
		logged = append(logged, Hash([]byte(req.System+"\n"+req.Messages[0].Content)))
		if _, err := f.Complete(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	if logged[0] == logged[1] {
		t.Fatal("different rendered prompts share a rendered hash")
	}
	for i, c := range f.Calls {
		sent := c.System + "\n" + c.Messages[0].Content
		if Hash([]byte(sent)) != logged[i] {
			t.Fatalf("call %d: logged hash does not match the bytes sent", i)
		}
		if !strings.HasPrefix(sent, p.Body) {
			t.Fatalf("call %d: system prompt is not the embedded file", i)
		}
	}
}
