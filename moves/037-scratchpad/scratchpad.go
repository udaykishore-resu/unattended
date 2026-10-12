// Package scratchpad gives an agent a per-run notes
// directory, and stubs old tool results in context.
//
// Move 37: scratchpad files over ever-longer context.
package scratchpad

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

type Notes struct{ root *os.Root }

func OpenNotes(dir, runID string) (*Notes, error) {
	d := filepath.Join(dir, runID)
	if err := os.MkdirAll(d, 0o750); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(d)
	if err != nil {
		return nil, err
	}
	return &Notes{root: r}, nil
}

func (n *Notes) Read(name string) (string, error) {
	f, err := n.root.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 8<<10))
	return string(b), err
}

// Write uses n.root.Create and rejects names that
// don't end in .md. Described in prose; minimal.
func (n *Notes) Write(name, text string) error {
	if !strings.HasSuffix(name, ".md") {
		return errors.New("note names must end in .md")
	}
	f, err := n.root.Create(name)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Close releases the root.
func (n *Notes) Close() error { return n.root.Close() }

func stub(msgs []llm.Message, done map[string]bool,
	note string) {
	for i := range msgs {
		m := &msgs[i]
		if m.Role == "tool" && done[m.ToolCallID] {
			m.Content = "result saved in " + note
		}
	}
}
