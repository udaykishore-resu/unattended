package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
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
	if want := hex.EncodeToString(sum[:6]); p.Version != want {
		t.Fatalf("version %s, want %s", p.Version, want)
	}
	if len(p.Version) != 12 || p.Body != string(disk) {
		t.Fatalf("prompt = %+v", p)
	}
	// Same bytes, same version: a hash cannot be forgotten like a number.
	again, _ := Load("classify")
	if again.Version != p.Version {
		t.Fatal("version not stable")
	}
}

func TestMissingPrompt(t *testing.T) {
	if _, err := Load("nope"); err == nil {
		t.Fatal("want an error for an unknown prompt")
	}
}
