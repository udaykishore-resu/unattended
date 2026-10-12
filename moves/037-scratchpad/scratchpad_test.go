package scratchpad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestNotesStayInTheRunDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "secret.md"), []byte("outside"), 0o600)
	n, err := OpenNotes(dir, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err := n.Write("rdx.md", "on-time 41 of 48 loads, from get_carrier_stats"); err != nil {
		t.Fatal(err)
	}
	got, err := n.Read("rdx.md")
	if err != nil || !strings.HasPrefix(got, "on-time 41") {
		t.Fatalf("%q %v", got, err)
	}
	// os.Root refuses paths that escape the directory.
	for _, bad := range []string{"../secret.md", "/etc/passwd", "a/../../secret.md"} {
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

func TestReadIsCapped(t *testing.T) {
	n, err := OpenNotes(t.TempDir(), "run-2")
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	n.Write("big.md", strings.Repeat("x", 20<<10))
	got, _ := n.Read("big.md")
	if len(got) != 8<<10 {
		t.Fatalf("read %d bytes, want cap %d", len(got), 8<<10)
	}
}

// Only the content changes: the tool message and its
// ToolCallID stay, so every call keeps its result.
func TestStubKeepsToolMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "c1"}, {ID: "c2"}}},
		{Role: "tool", ToolCallID: "c1", Content: strings.Repeat("raw ", 5000)},
		{Role: "tool", ToolCallID: "c2", Content: "keep me"},
	}
	stub(msgs, map[string]bool{"c1": true}, "rdx.md")
	if len(msgs) != 3 || msgs[1].ToolCallID != "c1" ||
		msgs[1].Content != "result saved in rdx.md" || msgs[2].Content != "keep me" {
		t.Fatalf("msgs = %+v", msgs[1:])
	}
}
