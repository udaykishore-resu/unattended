# Unattended: companion code

Runnable Go (and a small Python port) for every code example in *Unattended: 101 Field-Tested Moves for AI Automation You Can Trust to Run Alone* by Udaykishore Resu.

The book calls each technique a **Move**. Every move that shows Go code has its own directory here, with that code made runnable against the book's tiny `llm` client and a `FakeModel`, plus a test that demonstrates the move's claim. Nothing calls a real model: the tests run offline, in seconds.

## Quick start

```sh
git clone https://github.com/udaykishore-resu/unattended
cd unattended
go vet ./...
go test -race ./...                         # every move
go test -v ./moves/041-loop-guard           # one move, with test names
RUN_EVALS=1 go test ./moves/085-regression-suite
```

Python port:

```sh
cd python
python3 -m pip install pytest               # if you don't have it
python3 -m pytest
```

## Requirements

- **Go 1.26 or later** (`go.mod` says `go 1.26.0`). Tested with **Go 1.27.2**.
- **Standard library only.** No third-party modules, so `go test` needs no network.
- **Python 3.10 or later** for `python/` (tested with 3.10 and 3.13), plus `pytest`.

## Layout

```text
llm/                 the llm v2 client, exactly as in the book's front matter,
                     plus FakeModel and its tests
moves/NNN-name/      one directory per move with Go code; NNN is the move number
internal/sqlfake/    a tiny database/sql driver for tests (not in the book), so
                     moves that take a *sql.DB run without a database server
internal/psqltest/   runs a move's SQL constants verbatim in a real Postgres via
                     psql, for `-tags integration` tests (not in the book)
adapters/            README only: how a vendor adapter maps onto llm v2
python/              Python port of llm v2, FakeModel and five key moves
```

How a move directory maps to the book:

- The book's code is copied as printed, gofmt'd (the book prints 4-space indents; Go uses tabs).
- Where the book shows a fragment, the repo wraps it in the smallest function that runs it, and a comment marks where the book's code starts.
- Where the book names a helper only in prose (`parse`, `clip`, `askScores`, `verify`, ...), the repo implements it minimally, and a comment says so.
- Types the book describes in prose are defined next to the code. When one move uses another move's type (the `ToolError` from Move 32, the `Chunk` from Move 43), the directory carries a trimmed copy, so each move reads on its own. Move 83 imports Moves 56 and 89 for real, because its point is how those wrappers stack.
- Each `*_test.go` demonstrates the claim in the move's text, using the book's Halcyon Freight numbers where it can. A few examples: Move 41's guard trips on the second identical call and ends the run on the third; Move 52's key stays the same across prompt versions, so a replay of a finished step returns the saved result instead of resending; Move 89's budget charges every retry; Move 97's audit chain still verifies after a whole segment expires, and a failed audit write blocks the action; Move 98's two-key check refuses the same person twice, an author, or an edited proposal; Move 99 refuses absolute URLs and scrubs base64- and URL-encoded tokens.

Moves without a directory are prompts, SQL, checklists or process: their text in the book is the whole example.

## Move-to-directory map

### Part I — Pick the Right Work to Automate

| Move | Title | Code |
|-----:|-------|------|
| 1 | The five-question automation filter | — (prompt, SQL or checklist in the book) |
| 2 | Time the task before you automate it | [`moves/002-time-the-task`](moves/002-time-the-task) |
| 3 | Score tasks on frequency × pain × reversibility | [`moves/003-score-tasks`](moves/003-score-tasks) |
| 4 | Automate the handoff, not the whole job | [`moves/004-automate-handoff`](moves/004-automate-handoff) |
| 5 | Start from the exception queue, not the happy path | [`moves/005-exception-queue`](moves/005-exception-queue) (SQL; test runs it in Postgres via psql when `UNATTENDED_TEST_PSQL_URL` is set) |
| 6 | Write the SOP first; the SOP becomes the prompt | — (prompt, SQL or checklist in the book) |
| 7 | Design the kill switch before the feature | [`moves/007-kill-switch`](moves/007-kill-switch) |
| 8 | Calculate break-even in tokens, not hours | [`moves/008-break-even`](moves/008-break-even) |

### Part II — Prompts as Code

| Move | Title | Code |
|-----:|-------|------|
| 9 | Version prompts in git next to the code that calls them | [`moves/009-version-prompts`](moves/009-version-prompts) |
| 10 | Templated prompts with typed variables | [`moves/010-typed-templates`](moves/010-typed-templates) |
| 11 | Put the output schema before the instructions | [`moves/011-schema-first`](moves/011-schema-first) |
| 12 | Few-shot examples mined from your own failure log | [`moves/012-mined-examples`](moves/012-mined-examples) |
| 13 | A fixed skeleton: role → rules → examples → input | [`moves/013-prompt-skeleton`](moves/013-prompt-skeleton) |
| 14 | Force the format with native structured output | [`moves/014-native-structured-output`](moves/014-native-structured-output) |
| 15 | Keep the model's rationale in a field code never reads | [`moves/015-rationale-field`](moves/015-rationale-field) |
| 16 | Golden-file tests for prompts | [`moves/016-golden-files`](moves/016-golden-files) |
| 17 | Diff two prompt versions on the same 50 inputs | [`moves/017-prompt-diff`](moves/017-prompt-diff) |
| 18 | Lint your prompts: missing delimiters, vague verbs, conflicting rules | [`moves/018-prompt-lint`](moves/018-prompt-lint) |
| 19 | The deletion test: shrink a prompt without losing accuracy | [`moves/019-deletion-test`](moves/019-deletion-test) |

### Part III — Structured Output & Data Extraction

| Move | Title | Code |
|-----:|-------|------|
| 20 | Schemas and tool definitions as the extraction contract | [`moves/020-extraction-contract`](moves/020-extraction-contract) |
| 21 | Validate, repair once, then reject | [`moves/021-validate-repair-reject`](moves/021-validate-repair-reject) |
| 22 | Enums beat free text | [`moves/022-enums`](moves/022-enums) |
| 23 | Confidence plus evidence span for every field | [`moves/023-evidence-spans`](moves/023-evidence-spans) |
| 24 | Extract tables from PDFs page by page | [`moves/024-pdf-pages`](moves/024-pdf-pages) |
| 25 | Email-to-record pipelines | [`moves/025-email-to-record`](moves/025-email-to-record) |
| 26 | Invoice extraction with a totals cross-check | [`moves/026-totals-cross-check`](moves/026-totals-cross-check) |
| 27 | Dedup with embeddings plus an exact key | [`moves/027-dedup`](moves/027-dedup) |
| 28 | Normalize dates, currency and units outside the model | [`moves/028-normalize`](moves/028-normalize) |
| 29 | "Not present" is an answer: teach the model to abstain | [`moves/029-abstain`](moves/029-abstain) |

### Part IV — Tool Use & Agents That Don't Wander

| Move | Title | Code |
|-----:|-------|------|
| 30 | One tool, one verb | [`moves/030-one-tool-one-verb`](moves/030-one-tool-one-verb) |
| 31 | Tool descriptions are prompts: write them for a new hire | [`moves/031-tool-descriptions`](moves/031-tool-descriptions) |
| 32 | Return errors the model can act on | [`moves/032-actionable-errors`](moves/032-actionable-errors) |
| 33 | Reads run at once, writes wait for a yes | [`moves/033-gate-writes`](moves/033-gate-writes) |
| 34 | Dry-run mode for every write tool | [`moves/034-dry-run`](moves/034-dry-run) |
| 35 | Cap steps, tokens and wall clock per run | [`moves/035-run-caps`](moves/035-run-caps) |
| 36 | Plan, checkpoint, then execute | [`moves/036-plan-checkpoint`](moves/036-plan-checkpoint) |
| 37 | Scratchpad files over ever-longer context | [`moves/037-scratchpad`](moves/037-scratchpad) |
| 38 | Sub-agents for fan-out, one agent for judgment | [`moves/038-fan-out`](moves/038-fan-out) |
| 39 | Wrap internal APIs once as an MCP server, reuse everywhere | [`moves/039-mcp-backend`](moves/039-mcp-backend) |
| 40 | Least-privilege credentials per agent | [`moves/040-least-privilege`](moves/040-least-privilege) |
| 41 | Loop detection: same tool, same args, twice | [`moves/041-loop-guard`](moves/041-loop-guard) |

### Part V — RAG & Knowledge Automation

| Move | Title | Code |
|-----:|-------|------|
| 42 | Chunk by document structure, not character count | [`moves/042-structural-chunking`](moves/042-structural-chunking) |
| 43 | Contextual headers on every chunk | [`moves/043-contextual-headers`](moves/043-contextual-headers) |
| 44 | Hybrid search: keyword plus vectors | [`moves/044-hybrid-search`](moves/044-hybrid-search) |
| 45 | Retrieve 50, rerank to 5 | [`moves/045-rerank`](moves/045-rerank) |
| 46 | Cite or abstain | [`moves/046-cite-or-abstain`](moves/046-cite-or-abstain) |
| 47 | Freshness metadata and expiry dates | [`moves/047-freshness`](moves/047-freshness) |
| 48 | Index runbooks and closed tickets first | [`moves/048-runbooks-tickets`](moves/048-runbooks-tickets) |
| 49 | Rewrite vague queries before retrieval | [`moves/049-query-rewrite`](moves/049-query-rewrite) |
| 50 | Build the eval set from real search logs | [`moves/050-search-evals`](moves/050-search-evals) |

### Part VI — Workflow Orchestration

| Move | Title | Code |
|-----:|-------|------|
| 51 | Event-driven triggers over polling | [`moves/051-event-triggers`](moves/051-event-triggers) (SQL also runs in Postgres via psql with `-tags integration` and `UNATTENDED_TEST_PSQL_URL`) |
| 52 | Idempotency keys on every AI step | [`moves/052-idempotency-keys`](moves/052-idempotency-keys) (same; Python port too) |
| 53 | Durable execution for long workflows | [`moves/053-durable-execution`](moves/053-durable-execution) (same) |
| 54 | Queue plus bounded worker pool for batch jobs | [`moves/054-worker-pool`](moves/054-worker-pool) |
| 55 | Batch APIs for anything that can wait | [`moves/055-batch-api`](moves/055-batch-api) |
| 56 | Retries with jitter and a dead-letter queue humans read | [`moves/056-retry-dlq`](moves/056-retry-dlq) |
| 57 | Response caching and prompt caching for shared prefixes | [`moves/057-caching`](moves/057-caching) |
| 58 | Route by difficulty: small model first, escalate on low confidence | [`moves/058-difficulty-routing`](moves/058-difficulty-routing) |
| 59 | Human-in-the-loop as a queue, not an email | [`moves/059-review-queue`](moves/059-review-queue) (SQL; claim and decision races run in Postgres via psql with `-tags integration`) |
| 60 | Scheduled agents that file a run report | [`moves/060-run-report`](moves/060-run-report) |

### Part VII — Engineering & DevOps Automation

| Move | Title | Code |
|-----:|-------|------|
| 61 | PR review bot that only comments when it's sure | [`moves/061-pr-review-bot`](moves/061-pr-review-bot) |
| 62 | Generate a failing test from a bug report | [`moves/062-failing-test`](moves/062-failing-test) |
| 63 | Commit messages and changelogs from diffs | — (prompt, SQL or checklist in the book) |
| 64 | Dependency upgrades with a plain-English risk note | — (prompt, SQL or checklist in the book) |
| 65 | Log triage: cluster first, summarize second | [`moves/065-log-triage`](moves/065-log-triage) |
| 66 | Alert enrichment: attach the runbook and recent deploys | [`moves/066-alert-enrichment`](moves/066-alert-enrichment) |
| 67 | Auto-draft the incident timeline | [`moves/067-incident-timeline`](moves/067-incident-timeline) |
| 68 | Postmortem first draft from chat history and metrics | [`moves/068-postmortem-draft`](moves/068-postmortem-draft) |
| 69 | Review Terraform and Kubernetes manifests against policy | [`moves/069-plan-review`](moves/069-plan-review) |
| 70 | Flaky-test triage | [`moves/070-flaky-tests`](moves/070-flaky-tests) |
| 71 | Codebase Q&A over internal repos | [`moves/071-codebase-qa`](moves/071-codebase-qa) |
| 72 | On-call handoff summaries | [`moves/072-oncall-handoff`](moves/072-oncall-handoff) |

### Part VIII — Business & Personal Ops

| Move | Title | Code |
|-----:|-------|------|
| 73 | Inbox triage with exactly three labels | [`moves/073-inbox-triage`](moves/073-inbox-triage) |
| 74 | Meeting notes → action items → tracker | [`moves/074-action-items`](moves/074-action-items) |
| 75 | Scheduling assistant with hard constraints | [`moves/075-scheduling`](moves/075-scheduling) |
| 76 | One-page research briefs on leads, companies or candidates | [`moves/076-research-briefs`](moves/076-research-briefs) |
| 77 | Support ticket routing plus a draft reply | [`moves/077-ticket-routing`](moves/077-ticket-routing) |
| 78 | Contract clause extraction and red-flag notes | [`moves/078-contract-clauses`](moves/078-contract-clauses) |
| 79 | Weekly status report straight from your tools' APIs | [`moves/079-status-report`](moves/079-status-report) |
| 80 | One talk → five pieces of content | [`moves/080-talk-repurpose`](moves/080-talk-repurpose) |
| 81 | Call recording → summary → follow-up email | [`moves/081-call-followup`](moves/081-call-followup) |
| 82 | Personal knowledge base with a daily digest | [`moves/082-daily-digest`](moves/082-daily-digest) |

### Part IX — Evaluation, Observability & Cost

| Move | Title | Code |
|-----:|-------|------|
| 83 | Trace every call: prompt version, model, tokens, latency, outcome | [`moves/083-trace-every-call`](moves/083-trace-every-call) |
| 84 | LLM-as-judge with a rubric, and an eval for the judge | [`moves/084-llm-judge`](moves/084-llm-judge) |
| 85 | Prompt regression suite in CI | [`moves/085-regression-suite`](moves/085-regression-suite) |
| 86 | Shadow mode before cutover | [`moves/086-shadow-mode`](moves/086-shadow-mode) |
| 87 | Track human acceptance rate, not only accuracy | [`moves/087-acceptance-rate`](moves/087-acceptance-rate) |
| 88 | Cost per successful outcome, not cost per call | [`moves/088-cost-per-outcome`](moves/088-cost-per-outcome) (SQL; test needs Postgres via `UNATTENDED_TEST_PSQL_URL`) |
| 89 | Token budgets per workflow and per tenant | [`moves/089-token-budgets`](moves/089-token-budgets) |
| 90 | Drift alarms on output distributions | [`moves/090-drift-alarms`](moves/090-drift-alarms) |
| 91 | Sample a slice for human review, forever | [`moves/091-review-sampling`](moves/091-review-sampling) |
| 92 | A/B two models on live traffic safely | [`moves/092-ab-test`](moves/092-ab-test) |

### Part X — Safety, Security & Governance

| Move | Title | Code |
|-----:|-------|------|
| 93 | Treat every retrieved document as untrusted input | [`moves/093-taint`](moves/093-taint) |
| 94 | Separate instructions from data structurally | [`moves/094-fences`](moves/094-fences) |
| 95 | Redact PII before the model, re-hydrate after | [`moves/095-redaction`](moves/095-redaction) |
| 96 | Allow-lists for actions, not deny-lists | [`moves/096-allow-lists`](moves/096-allow-lists) |
| 97 | Audit-log every agent action | [`moves/097-audit-log`](moves/097-audit-log) |
| 98 | Approval tiers by blast radius | [`moves/098-approval-tiers`](moves/098-approval-tiers) |
| 99 | Secrets never enter the context window | [`moves/099-secrets`](moves/099-secrets) |
| 100 | Write the "how this automation fails" page before launch | [`moves/100-failure-page`](moves/100-failure-page) (the CI check the move describes; the page template is in the book) |

### Part XI — Capstone

| Move | Title | Code |
|-----:|-------|------|
| 101 | From SOP to production: one automation, end to end | [`moves/101-capstone`](moves/101-capstone): the prompt's rules enforced in Go, and the failure policy |

## Python port

`python/` mirrors the `llm` v2 types as dataclasses (`Request`, `Message`, `Tool`, `ToolCall`, `StopReason`, `Response`, `Usage`) and `FakeModel`, plus five moves the editor picked as the core of the book:

| Module | Moves |
|--------|-------|
| `move021_repair.py` | 21: validate, repair once, then reject |
| `move033_gate.py` | 33 and 34: tool gating, with a dry-run plan per write |
| `move041_loop_guard.py` | 41: loop detection |
| `move052_idempotency.py` | 52: idempotency keys (on `sqlite3`, standard library) |
| `move089_budget.py` | 89: token budgets per workflow and tenant |

Names follow Python conventions (`max_tokens`, `tool_call_id`); behavior and tests match the Go versions.

## Errata

Found a mistake in the book or the code? Please open a [GitHub issue](https://github.com/udaykishore-resu/unattended/issues) with the move number and what you expected. Confirmed errata are fixed here first and listed in the issue tracker.

## Skills

What this code demonstrates, move by move:

- **Reliable LLM integration in Go**: a vendor-neutral client, typed stop reasons, structured output with native schema modes or forced tool calls, and strict decoding.
- **Validation and grounding**: validate-repair-reject, enums, evidence spans checked against the source, citation checks, totals cross-checks, abstention.
- **Agent safety**: one-verb tools, actionable tool errors, read/write gating with approval queues, dry-run plans, loop guards, step/token/wall-clock caps, taint tracking, allow-lists, least-privilege credentials, secret scrubbing.
- **Workflow reliability**: idempotency keys with leases, durable step journals, bounded worker pools, retries with full jitter and permanent errors, batch reconciliation, prompt and response caching.
- **Retrieval (RAG)**: structural chunking, contextual headers, reciprocal rank fusion, reranking, freshness decay, query rewriting, recall@k and MRR.
- **Evaluation and observability**: golden-file prompt tests, prompt diffs, per-label regression floors, shadow mode, acceptance tracking, PSI drift alarms, stratified review sampling, unit-level A/B splits, call tracing.
- **Governance**: tamper-evident audit logs with per-segment hash chains, approval tiers by blast radius, PII redaction and re-hydration.
- **Testing without a model**: a recording `FakeModel`, a stdlib `database/sql` fake, `httptest`, and race-detector-clean concurrency.

## GitHub topics

`ai-automation` `llm` `agents` `go` `golang` `reliability` `book-companion` `rag` `evals` `idempotency` `prompt-engineering` `python`

## License

MIT. See [LICENSE](LICENSE). Copyright (c) 2026 Udaykishore Resu.
