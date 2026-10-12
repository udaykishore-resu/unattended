// Package scratchpad gives an agent a per-run notes
// directory, and stubs old tool results in context.
//
// Move 37: scratchpad files over ever-longer context.
package scratchpad

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

type Notes struct{ root *os.Root }

func OpenNotes(dir, runID string) (*Notes, error) {
	d := filepath.Join(dir, runID)
	if err := os.Mkdir(d, 0o750); err != nil {
		return nil, err // exists: not a fresh run
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
	b, err := io.ReadAll(io.LimitReader(f, maxNote))
	return string(b), err
}

// maxNote is both the write limit and the read cap.
const maxNote = 8 << 10

// Write uses n.root.Create, accepts only .md names,
// refuses notes over maxNote with an error the model
// can act on, and syncs before it reports success.
// Described in prose; minimal.
func (n *Notes) Write(name, text string) error {
	if !strings.HasSuffix(name, ".md") {
		return errors.New("note names must end in .md")
	}
	if len(text) > maxNote {
		return fmt.Errorf("note is %d bytes, max %d: "+
			"write one short note per item",
			len(text), maxNote)
	}
	f, err := n.root.Create(name)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Close releases the root.
func (n *Notes) Close() error { return n.root.Close() }

func stub(n *Notes, msgs []llm.Message,
	done map[string]bool, note string) error {
	if s, err := n.Read(note); err != nil || s == "" {
		return fmt.Errorf("note %q not saved", note)
	}
	for i := range msgs {
		m := &msgs[i]
		if m.Role == "tool" && done[m.ToolCallID] {
			m.Content = "result saved in " + note
		}
	}
	return nil
}
