import pytest

from unattended.llm import (FakeModel, Message, NoResponsesError, Request,
                            Response, StopReason, ToolCall, Usage)


def test_replays_in_order_then_errors():
    f = FakeModel(responses=[Response(text="one"), Response(text="two")])
    assert f.complete(Request()).text == "one"
    assert f.complete(Request()).text == "two"
    with pytest.raises(NoResponsesError):
        f.complete(Request())
    assert len(f.calls) == 3


def test_fills_stop_reason():
    f = FakeModel(responses=[
        Response(text="done"),
        Response(tool_calls=[ToolCall("c1", "get_load")]),
        Response(text="cut", stop_reason=StopReason.MAX_TOKENS),
    ])
    got = [f.complete(Request()).stop_reason for _ in range(3)]
    assert got == [StopReason.END, StopReason.TOOL_USE, StopReason.MAX_TOKENS]


def test_copies_history():
    f = FakeModel(responses=[Response(), Response()])
    req = Request(messages=[Message("user", "hi")])
    f.complete(req)
    req.messages.append(Message("assistant", "x"))  # an agent loop appends
    f.complete(req)
    assert len(f.calls[0].messages) == 1


def test_err():
    f = FakeModel(err=TimeoutError("boom"))
    with pytest.raises(TimeoutError):
        f.complete(Request())


def test_usage_has_four_fields():
    u = Usage(1, 2, 3, 4)
    assert (u.input_tokens, u.cache_read_tokens,
            u.cache_write_tokens, u.output_tokens) == (1, 2, 3, 4)
