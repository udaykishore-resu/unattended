// Package allowlist gates every tool call on its
// name and its arguments. Absent means denied.
//
// Move 96: allow-lists for actions, not deny-lists.
package allowlist

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/udaykishore-resu/unattended/llm"
)

type Check func(in json.RawMessage) error

// Allowed is the whole list. Absent means denied.
var Allowed = map[string]Check{
	"get_shipment": nil, // no arg check; authz in tool
	"add_note":     maxLen("text", 2000),
	"set_status": oneOf("status",
		"in_transit", "delivered", "delayed"),
}

// Gate runs in the executor before any tool runs.
func Gate(name string, in json.RawMessage) error {
	chk, ok := Allowed[name]
	if !ok {
		return fmt.Errorf("tool %q not allowed", name)
	}
	if chk == nil {
		return nil
	}
	return chk(in)
}

// oneOf and maxLen are a dozen lines each:
// unmarshal, read one field, compare.

func field(in json.RawMessage, name string) (string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(in, &m); err != nil {
		return "", fmt.Errorf("bad input: %w", err)
	}
	var s string
	if err := json.Unmarshal(m[name], &s); err != nil {
		return "", fmt.Errorf("%s: want a string", name)
	}
	return s, nil
}

func oneOf(name string, vals ...string) Check {
	return func(in json.RawMessage) error {
		v, err := field(in, name)
		if err != nil {
			return err
		}
		if !slices.Contains(vals, v) {
			return fmt.Errorf("%s %q not allowed", name, v)
		}
		return nil
	}
}

func maxLen(name string, n int) Check {
	return func(in json.RawMessage) error {
		v, err := field(in, name)
		if err != nil {
			return err
		}
		if len(v) > n {
			return fmt.Errorf("%s over %d bytes", name, n)
		}
		return nil
	}
}

// Tool is a handler the executor may run.
type Tool func(ctx context.Context,
	in json.RawMessage) (string, error)

// Executor is the one path from a ToolCall to a
// handler. Tools may come from code, plugins or an
// MCP server; registering one does not allow it.
type Executor struct {
	Tools map[string]Tool
}

// Do runs Gate first. A denial or a check error,
// such as a timeout, never reaches the handler.
func (e *Executor) Do(ctx context.Context,
	c llm.ToolCall) (string, error) {
	if err := Gate(c.Name, c.Input); err != nil {
		return "", fmt.Errorf("denied: %w", err)
	}
	run, ok := e.Tools[c.Name]
	if !ok {
		return "", fmt.Errorf("tool %q not found", c.Name)
	}
	return run(ctx, c.Input)
}
