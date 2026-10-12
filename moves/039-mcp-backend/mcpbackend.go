// Package mcpbackend is the plain Go interface that
// sits behind an MCP server, and the agent-side
// filter that keeps only tools its registry knows.
//
// Move 39: wrap internal APIs once as an MCP server.
// No MCP SDK here: the book says to use an official
// SDK for framing and transports; this is the part
// you own.
package mcpbackend

import (
	"context"
	"encoding/json"

	"github.com/udaykishore-resu/unattended/llm"
)

type Backend interface {
	Tools() []llm.Tool
	Call(ctx context.Context, name string,
		args json.RawMessage) (string, error)
}

func gateTools(listed []llm.Tool,
	reg map[string]Entry) []llm.Tool {
	var out []llm.Tool
	for _, t := range listed {
		if _, ok := reg[t.Name]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Entry is the registry row from Move 33: kind and
// egress come from here, never from the server.
type Entry struct {
	Write  bool
	Egress bool
}
