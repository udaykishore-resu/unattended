// Package llm is the tiny model client used in
// every example in this book.
package llm

import (
	"context"
	"encoding/json"
)

// Model is anything that can complete a request.
type Model interface {
	Complete(ctx context.Context, r Request) (
		Response, error)
}

type Request struct {
	System    string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
	// Schema is an optional JSON Schema for output.
	Schema json.RawMessage
	// CacheSystem marks System+Tools as a
	// cacheable prefix.
	CacheSystem bool
}

// Message is one turn. Assistant turns may carry
// ToolCalls; a tool result is Role "tool" with
// ToolCallID set to the call it answers.
type Message struct {
	Role       string // "user", "assistant", "tool"
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Tool struct {
	Name, Description string
	InputSchema       json.RawMessage
}

type ToolCall struct {
	ID, Name string
	Input    json.RawMessage
}

type StopReason string

const (
	StopEnd       StopReason = "end"
	StopMaxTokens StopReason = "max_tokens"
	StopToolUse   StopReason = "tool_use"
	StopRefusal   StopReason = "refusal"
	StopOther     StopReason = "other"
)

type Response struct {
	Text       string
	ToolCalls  []ToolCall
	Usage      Usage
	StopReason StopReason
}

// Usage splits input so cost math works on every
// vendor. Total input = Input + CacheRead +
// CacheWrite.
type Usage struct {
	InputTokens      int // uncached input
	CacheReadTokens  int
	CacheWriteTokens int
	OutputTokens     int
}
