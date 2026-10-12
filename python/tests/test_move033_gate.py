import pytest

from unattended.llm import ToolCall
from unattended.move033_gate import (Entry, Executor, Kind, ToolError,
                                     plan_set_status, set_status)


class MemQueue:
    def __init__(self):
        self.got = []

    def propose(self, p):
        self.got.append(p)
        return f"prop-{len(self.got)}"


class MemStore:
    def __init__(self):
        self.loads = {"HF-1": {"id": "HF-1", "status": "in_transit"}}
        self.writes = 0

    def get_load(self, load_id):
        return dict(self.loads[load_id])

    def set_status(self, load_id, status):
        self.writes += 1
        self.loads[load_id]["status"] = status


def executor(store):
    import json

    def apply(inp):
        d = json.loads(inp)
        return set_status(store, d["load_id"], d["status"], dry=False)

    def plan(inp):
        d = json.loads(inp)
        return plan_set_status(store, d["load_id"], d["status"])

    return Executor(
        run_id="run-7", queue=MemQueue(),
        tools={"get_load": Entry(Kind.READ, lambda _: "ok"),
               "fetch_url": Entry(Kind.READ, lambda _: "ok", egress=True),
               "set_status": Entry(Kind.WRITE, apply)},
        planners={"set_status": plan})


def test_reads_run_writes_queue_with_plan():
    store = MemStore()
    ex = executor(store)
    assert ex.do(ToolCall("c1", "get_load")) == "ok"
    out = ex.do(ToolCall("c2", "set_status",
                         '{"load_id":"HF-1","status":"delivered"}'))
    assert "Do not call it again" in out
    assert store.writes == 0  # nothing changed
    p = ex.queue.got[0]
    assert p.run_id == "run-7" and p.plan.changes == {
        "status": ("in_transit", "delivered")}


def test_tainted_run_loses_egress_and_unknown_is_refused():
    ex = executor(MemStore())
    ex.tainted = True
    with pytest.raises(ToolError) as e:
        ex.do(ToolCall("c1", "fetch_url", '{"url":"x"}'))
    assert e.value.code == "tainted"
    assert ex.do(ToolCall("c2", "get_load")) == "ok"  # closed read still works
    with pytest.raises(ToolError) as e:
        ex.do(ToolCall("c3", "cancel_load"))
    assert e.value.code == "unknown_tool"


def test_dry_run_changes_nothing():
    store = MemStore()
    out = set_status(store, "HF-1", "delivered", dry=True)
    assert out.startswith("DRY RUN, nothing changed.")
    assert store.writes == 0 and store.loads["HF-1"]["status"] == "in_transit"


def test_noop_is_flagged_and_live_run_applies():
    store = MemStore()
    assert plan_set_status(store, "HF-1", "in_transit").notes == ["no-op: same status"]
    set_status(store, "HF-1", "delivered", dry=False)
    assert store.writes == 1
