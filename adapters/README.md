# Writing a vendor adapter for `llm` v2

Every move in this repo talks to a model through one interface:

```go
type Model interface {
    Complete(ctx context.Context, r Request) (Response, error)
}
```

An **adapter** is the one place that knows a vendor exists. It implements
`Complete` by translating `llm.Request` into the vendor's wire format and the
vendor's reply back into `llm.Response`. Nothing outside the adapter ever sees a
vendor string, so retries, budgets, tracing and every move stay vendor-neutral.

This directory deliberately holds no adapter code: vendor SDKs change faster
than a book, and the repo is stdlib-only. Write yours against your vendor's
current SDK or HTTP API, and test it with the contract tests at the end of this
page.

## The shape of an adapter

```text
llm.Request ──► build vendor request ──► call vendor ──► map reply ──► llm.Response
                 (system, turns, tools,                 (text, tool calls,
                  schema, cache marks)                   usage, stop reason)
```

Keep it thin. Retries, budgets, caching and logging go in wrappers that also
implement `llm.Model`, stacked in the book's order (Move 83):

```go
m := Retrying{Next: Budgeted{Next: Traced{Next: adapter, ...}}}
```

So the adapter itself makes **one attempt** and returns. If the vendor SDK has
its own retry loop, turn it off; retrying in two layers multiplies attempts
(Move 56).

## 1. Stop reasons

Map every vendor stop value onto exactly one `llm.StopReason`. Code in the moves
switches on these constants and never on a vendor spelling.

| `llm.StopReason`  | Means                                    | Vendor values look like                         |
|-------------------|------------------------------------------|-------------------------------------------------|
| `StopEnd`         | the model finished its answer            | "stop", "end_turn", "STOP", "completed"          |
| `StopMaxTokens`   | output hit `MaxTokens`; reply is cut off | "length", "max_tokens", "MAX_TOKENS"             |
| `StopToolUse`     | the reply ends in one or more tool calls | "tool_calls", "tool_use", "function_call"        |
| `StopRefusal`     | the model or a safety layer declined     | "refusal", "content_filter", "SAFETY"            |
| `StopOther`       | anything else, including values you have never seen | stop sequences, "recitation", new values |

Rules:

- **Unknown values map to `StopOther`, never to `StopEnd`.** Moves treat anything
  but `StopEnd` (or `StopToolUse` for tool turns) as "do not trust this output"
  (Moves 4, 14, 24, 55). A new vendor value must fail closed.
- **A refusal is a stop reason, not an error.** Return it in `Response` with a
  nil error, so callers can route it to a person instead of retrying (Move 14).
  Some vendors put refusals in a separate field or return an empty choice with a
  filter flag; check all of them.
- **Tool calls win.** If the reply carries tool calls, report `StopToolUse` even
  when the vendor also says it "stopped".
- **Truncated tool arguments are `StopMaxTokens`.** Half a JSON argument must
  never execute (Move 32).

## 2. Tool turns

`llm` models a tool loop with three message shapes:

| `llm.Message`                                | Vendor equivalent                                        |
|----------------------------------------------|----------------------------------------------------------|
| `Role: "user"`, `Content`                    | a user message / user content block                      |
| `Role: "assistant"`, `Content`, `ToolCalls`  | an assistant message with text plus tool-call entries    |
| `Role: "tool"`, `ToolCallID`, `Content`      | a tool-result entry that names the call it answers      |

Things adapters get wrong:

- **Every call gets exactly one result.** The loop in Move 32 appends the
  assistant turn with all its `ToolCalls`, then one `"tool"` message per call.
  Some vendors want results as separate messages with a "tool" role; others
  want all results for one assistant turn grouped into a single user turn as
  tool-result blocks. Group consecutive `"tool"` messages when your vendor
  needs it, and keep their order.
- **Preserve call IDs exactly.** `ToolCallID` must round-trip byte for byte. If a
  vendor returns calls without IDs, synthesize stable ones (for example,
  `call_<turn>_<index>`) and remember the mapping for the result turn.
- **Arguments are raw JSON.** Put the vendor's argument object (or string) into
  `ToolCall.Input` as `json.RawMessage`. If the vendor sends arguments as a JSON
  string, decode the string once, so `Input` holds the object, not a quoted
  string.
- **Tool errors.** If your vendor has an "is error" flag on tool results, set it
  when the content is a `ToolError` JSON body (Move 32); otherwise send the body
  as plain content.
- **Tool definitions.** `llm.Tool{Name, Description, InputSchema}` maps directly.
  Sort tools by name before sending, so a cached prefix stays stable (Move 57).

## 3. Structured output (`Request.Schema`)

When `Schema` is set, use the vendor's native structured-output or JSON-schema
mode, so the reply comes back in `Response.Text`. If the vendor only supports
schemas through tools, send one synthetic tool with that schema and **force**
it; return the arguments in `ToolCalls[0].Input` (Move 14 reads both places).

When a request carries exactly one tool and no schema, Halcyon's adapter forces
that tool (Move 20). Do the same, or document that you don't.

Do not fall back to assistant-turn prefill unless the vendor has no schema
mode at all: several current APIs reject a final assistant turn. And strict
modes may skip `maxLength`, `minimum` or `pattern`; the moves enforce those in
Go anyway (Moves 15, 21, 38).

Give each adapter a test that asserts the outgoing request uses native mode
when `Schema` is set. Dropping the schema silently is the classic regression.

## 4. Cache marks and cache usage

`Request.CacheSystem` asks the adapter to mark `System` plus `Tools` as a
reusable prefix:

- Vendors with **explicit** cache breakpoints: put the breakpoint after the
  last tool definition (or after the system prompt if there are no tools).
- Vendors with **automatic** prefix caching: there is nothing to send; the flag
  is a no-op, but keep the prefix byte-stable.
- Vendors with **named cache objects**: create or reuse the cached object keyed
  by a hash of `System` plus the sorted tools.

`llm.Usage` splits input so cost math works on every vendor:

```text
total input = InputTokens + CacheReadTokens + CacheWriteTokens
```

| `llm.Usage` field   | Fill it with                                                  |
|---------------------|---------------------------------------------------------------|
| `InputTokens`       | **uncached** input only                                       |
| `CacheReadTokens`   | input served from the cache                                   |
| `CacheWriteTokens`  | input written to the cache on this call (0 if not reported)   |
| `OutputTokens`      | output tokens, including any reasoning tokens you are billed for |

The trap: some vendors report a total input count that **includes** cached
tokens, plus a separate "cached" count. Subtract, so `InputTokens` is the
uncached part only. Others report the three numbers separately; copy them.
Never leave cached tokens out, or budgets (Moves 35, 89) and cost per outcome
(Move 88) will undercount a cache-heavy workflow.

With streaming, many vendors send usage only in the final event. Read the
stream to the end before returning, or `Usage` will be zeros (Move 83).

## 5. Errors

- Transport errors, timeouts, 429 and 5xx: return them as plain errors; the
  retry wrapper (Move 56) decides. If the vendor sends a retry-after hint, wrap
  it in an error type the retry layer can read.
- 400-class request errors (bad schema, context too long, invalid tool turn):
  wrap them in `Permanent` (Move 56), so they are never retried.
- If the vendor billed tokens on a failed call, still report them: the budget
  wrapper charges whatever `Usage` you return (Move 89).
- Never put the API key, request headers or full prompt into an error string.
  Errors reach logs and, through `toolResult`, can reach a model (Moves 32, 99).

## 6. Contract tests for an adapter

Run these against recorded vendor responses (fixtures you capture once and
check in), not the live API:

1. A plain reply maps to `StopEnd` with `Text` set.
2. A length-limited reply maps to `StopMaxTokens`.
3. A refusal maps to `StopRefusal` with a nil error.
4. An unknown stop value maps to `StopOther`.
5. A tool-call reply maps to `StopToolUse`, with IDs, names and raw-JSON inputs.
6. A three-turn tool loop (assistant calls, two tool results, final answer)
   serializes in the vendor's required grouping and order.
7. `Schema` set: the outgoing request uses native structured output (or a
   forced tool).
8. `CacheSystem` set: the outgoing request carries the cache mark (or the
   adapter documents that the vendor caches automatically).
9. Usage with cached input: `InputTokens + CacheReadTokens + CacheWriteTokens`
   equals the vendor's total input, and `InputTokens` excludes cached tokens.
10. A 400 response comes back wrapped as `Permanent`; a 429 does not.

`llm.FakeModel` stands in for an adapter everywhere else in this repo, so the
moves can be tested with no network.
