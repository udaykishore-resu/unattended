"""Moves 33 and 34: reads run at once, writes wait for a yes; every write
tool has a dry-run plan half.

The executor runs reads immediately and turns every write into a queued
proposal. A tainted run loses egress tools. Each write handler is split into
plan (reads and computes old -> new) and apply; dry-run calls only plan.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Callable, Protocol

from .llm import ToolCall


class Kind(Enum):
    READ = "read"
    WRITE = "write"


class ToolError(Exception):
    """Move 32's shape: a failure the model can act on."""

    def __init__(self, code: str, msg: str, fix: str, retry: bool = False):
        super().__init__(f"{code}: {msg}")
        self.code, self.msg, self.fix, self.retry = code, msg, fix, retry

    def to_json(self) -> str:
        return json.dumps({"code": self.code, "msg": self.msg,
                           "fix": self.fix, "retry": self.retry})


@dataclass
class Entry:
    kind: Kind
    run: Callable[[str], str]
    egress: bool = False  # input reaches another system


@dataclass
class Proposal:
    run_id: str
    call_id: str
    tool: str
    input: str
    plan: Any = None  # the dry-run plan rides with the proposal


class Queue(Protocol):
    def propose(self, p: Proposal) -> str: ...


@dataclass
class Executor:
    run_id: str
    tools: dict[str, Entry]
    queue: Queue
    tainted: bool = False
    planners: dict[str, Callable[[str], "Plan"]] = field(default_factory=dict)

    def do(self, c: ToolCall) -> str:
        ent = self.tools.get(c.name)
        if ent is None:
            raise ToolError("unknown_tool", "no such tool for this agent",
                            "do not retry; report it and stop")
        if self.tainted and ent.egress:
            raise ToolError("tainted", "this run read outside text",
                            "do not retry; finish without this tool")
        if ent.kind is Kind.READ:
            return ent.run(c.input)
        plan = self.planners[c.name](c.input) if c.name in self.planners else None
        pid = self.queue.propose(Proposal(self.run_id, c.id, c.name, c.input, plan))
        return f"queued for approval as {pid}. Do not call it again."


# --- Move 34: plan and apply ---

@dataclass
class Plan:
    tool: str
    target: str
    changes: dict[str, tuple[Any, Any]]
    notes: list[str] = field(default_factory=list)


class Store(Protocol):
    def get_load(self, load_id: str) -> dict: ...
    def set_status(self, load_id: str, status: str) -> None: ...


def plan_set_status(db: Store, load_id: str, status: str) -> Plan:
    load = db.get_load(load_id)
    p = Plan("set_status", load["id"], {"status": (load["status"], status)})
    if load["status"] == status:
        p.notes.append("no-op: same status")
    return p


def set_status(db: Store, load_id: str, status: str, dry: bool) -> str:
    """The live path builds the plan first, so dry and real runs share the
    same checks. `dry` comes from config per run, never from the model."""
    p = plan_set_status(db, load_id, status)
    old = p.changes["status"][0]
    summary = f"set_status {p.target}: status {old} -> {status}"
    if dry:
        return "DRY RUN, nothing changed. " + summary
    if db.get_load(p.target)["status"] != old:
        raise RuntimeError("stale plan")
    db.set_status(p.target, status)
    return summary
