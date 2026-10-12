import pytest

from unattended.llm import ToolCall
from unattended.move033_gate import ToolError
from unattended.move041_loop_guard import LoopError, LoopGuard


def test_trips_on_second_identical_call():
    g = LoopGuard()
    c = ToolCall("c1", "add_note", '{"load_id":"HF-90877","text":"carrier confirmed ETA"}')
    g.check(c)
    with pytest.raises(ToolError) as e:
        g.check(c)
    assert e.value.code == "repeat_call"
    with pytest.raises(LoopError):
        g.check(c)


def test_key_order_ignored():
    g = LoopGuard()
    g.check(ToolCall("a", "get_load", '{"load_id":"HF-1","fields":["eta"]}'))
    with pytest.raises(ToolError):
        g.check(ToolCall("b", "get_load", '{ "fields": ["eta"], "load_id": "HF-1" }'))


def test_different_args_pass():
    g = LoopGuard()
    for i in range(3):
        g.check(ToolCall(str(i), "get_load", f'{{"load_id":"HF-{i}"}}'))


def test_third_identical_call_ends_the_run_with_clear_message():
    g = LoopGuard()
    c = ToolCall("c1", "get_load", '{"load_id":"HF-1"}')
    g.check(c)
    with pytest.raises(ToolError):
        g.check(c)
    with pytest.raises(LoopError, match="third identical call"):
        g.check(c)


def test_big_numbers_stay_distinct():
    g = LoopGuard()
    g.check(ToolCall("a", "get_load", '{"pro":12345678901234567}'))
    g.check(ToolCall("b", "get_load", '{"pro":12345678901234568}'))
