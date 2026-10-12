// Package prompts embeds prompt files and names
// each by a hash of its bytes.
//
// Move 9: version prompts in git next to the code
// that calls them.
package prompts

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
)

//go:embed *.txt
var files embed.FS

type Prompt struct{ Name, Body, Version string }

func Load(name string) (Prompt, error) {
	b, err := files.ReadFile(name + ".txt")
	if err != nil {
		return Prompt{}, err
	}
	return Prompt{name, string(b), Hash(b)}, nil
}

// Hash is the full SHA-256 of the exact bytes:
// a template file, or a rendered prompt as sent.
func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
