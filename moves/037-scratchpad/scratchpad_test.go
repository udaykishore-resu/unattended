package scratchpad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func open(t *testing.T, dir, run string) *Notes {
	t.Helper()
	n, err := OpenNotes(dir, run)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	return n
}

func TestNotesStayInTheRunDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "secret.md"), []byte("outside"), 0o600)
	n := open(t, dir, "run-1")
	if err := n.Write("rdx.md", "on-time 41 of 48 loads, from get_carrier_stats"); err != nil {
		t.Fatal(err)
	}
	got, err := n.Read("rdx.md")
	if err != nil || !strings.HasPrefix(got, "on-time 41") {
		t.Fatalf("%q %v", got, err)
	}
	// os.Root refuses paths that escape the directory.
	for _, bad := range []string{"../secret.md", "/etc/passwd", "a/../../secret.md", "../run-1/../secret.md"} {
		if _, err := n.Read(bad); err == nil {
			t.Errorf("read %q escaped the root", bad)
		}
	}
	if err := n.Write("../x.md", "no"); err == nil {
		t.Error("write escaped the root")
	}
	if err := n.Write("notes.txt", "no"); err == nil {
		t.Error("non-.md name accepted")
	}
}

// Symlinks that point outside the run directory are
// refused, relative or absolute, for reads and
// writes. (A symlink can't be planted by the model:
// the tools only write plain files. This covers a
// directory tampered with from outside.)
func TestSymlinksCannotEscape(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.md")
	os.WriteFile(secret, []byte("outside"), 0o600)
	n := open(t, dir, "run-1")
	run := filepath.Join(dir, "run-1")
	os.Symlink("../secret.md", filepath.Join(run, "rel.md"))
	os.Symlink(secret, filepath.Join(run, "abs.md"))
	os.Symlink(dir, filepath.Join(run, "up"))
	for _, name := range []string{"rel.md", "abs.md", "up/secret.md"} {
		if s, err := n.Read(name); err == nil {
			t.Errorf("read %q escaped via symlink: %q", name, s)
		}
		if err := n.Write(name, "overwrite"); err == nil {
			t.Errorf("write %q escaped via symlink", name)
		}
	}
	if b, _ := os.ReadFile(secret); string(b) != "outside" {
		t.Fatalf("secret changed: %q", b)
	}
}

func TestReadIsCapped(t *testing.T) {
	dir := t.TempDir()
	n := open(t, dir, "run-2")
	// A note planted past the write limit is still cut at the read cap.
	os.WriteFile(filepath.Join(dir, "run-2", "big.md"), []byte(strings.Repeat("x", 20<<10)), 0o600)
	got, _ := n.Read("big.md")
	if len(got) != maxNote {
		t.Fatalf("read %d bytes, want cap %d", len(got), maxNote)
	}
}

func TestOversizedWriteRefused(t *testing.T) {
	n := open(t, t.TempDir(), "run-3")
	err := n.Write("giant.md", strings.Repeat("x", maxNote+1))
	if err == nil || !strings.Contains(err.Error(), "one short note per item") {
		t.Fatalf("got %v", err)
	}
	if _, err := n.Read("giant.md"); err == nil {
		t.Fatal("oversized note was created")
	}
	if err := n.Write("ok.md", strings.Repeat("x", maxNote)); err != nil {
		t.Fatalf("note at the limit refused: %v", err)
	}
}

// Every run starts empty: a second OpenNotes for the
// same run ID fails instead of reusing old notes.
func TestRunStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	n := open(t, dir, "run-4")
	n.Write("rdx.md", "yesterday's numbers")
	if _, err := OpenNotes(dir, "run-4"); err == nil {
		t.Fatal("reopened an existing run directory")
	}
}

func msgs() []llm.Message {
	return []llm.Message{
		{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "c1"}, {ID: "c2"}}},
		{Role: "tool", ToolCallID: "c1", Content: strings.Repeat("raw ", 5000)},
		{Role: "tool", ToolCallID: "c2", Content: "keep me"},
	}
}

// Only the content changes: the tool message and its
// ToolCallID stay, so every call keeps its result.
func TestStubKeepsToolMessages(t *testing.T) {
	n := open(t, t.TempDir(), "run-5")
	if err := n.Write("rdx.md", "on-time 41 of 48"); err != nil {
		t.Fatal(err)
	}
	m := msgs()
	if err := stub(n, m, map[string]bool{"c1": true}, "rdx.md"); err != nil {
		t.Fatal(err)
	}
	if len(m) != 3 || m[1].ToolCallID != "c1" ||
		m[1].Content != "result saved in rdx.md" || m[2].Content != "keep me" {
		t.Fatalf("msgs = %+v", m[1:])
	}
}

// No note, an empty note, or a bad name: stub
// refuses and the raw results stay in context.
func TestStubNeedsASavedNote(t *testing.T) {
	n := open(t, t.TempDir(), "run-6")
	n.Write("empty.md", "")
	for _, note := range []string{"missing.md", "empty.md", "../secret.md"} {
		m := msgs()
		if err := stub(n, m, map[string]bool{"c1": true}, note); err == nil {
			t.Errorf("stubbed against %q", note)
		}
		if !strings.HasPrefix(m[1].Content, "raw ") {
			t.Errorf("raw result replaced for %q", note)
		}
	}
}
