import pytest

from unattended.llm import FakeModel, Message, Request, Response, ToolCall
from unattended.move021_repair import (NoToolCallError, RejectedError,
                                       extract)

GOOD = ('{"shipper_ref":"A","origin_zip":"45402","dest_zip":"60601",'
        '"pickup_date":"3/12","weight_lb":42000}')
BAD = GOOD.replace('"45402"', '"4540"')


def call(cid, inp):
    return Response(tool_calls=[ToolCall(cid, "save_tender", inp)])


def test_code_fixes_zip_plus_4_without_a_call():
    f = FakeModel(responses=[call("c1", GOOD.replace('"45402"', '"45402-1123"'))])
    t = extract(f, Request())
    assert t.origin_zip == "45402" and len(f.calls) == 1


def test_one_repair_then_pass():
    f = FakeModel(responses=[call("c1", BAD), call("c2", GOOD)])
    t = extract(f, Request(messages=[Message("user", "email")]))
    assert t.origin_zip == "45402"
    msgs = f.calls[1].messages
    assert [m.role for m in msgs] == ["user", "assistant", "tool"]
    assert msgs[1].tool_calls and msgs[2].tool_call_id == "c1"
    assert "'4540'" in msgs[2].content and "use null" in msgs[2].content


def test_reject_after_one_repair():
    f = FakeModel(responses=[call("c1", BAD), call("c2", BAD)])
    with pytest.raises(RejectedError):
        extract(f, Request())
    assert len(f.calls) == 2  # never a third try


def test_null_is_an_answer():
    f = FakeModel(responses=[call("c1", GOOD.replace('"45402"', "null"))])
    assert extract(f, Request()).origin_zip is None


def test_prose_reply_goes_to_review():
    f = FakeModel(responses=[Response(text="see attached")])
    with pytest.raises(NoToolCallError):
        extract(f, Request())


def test_unknown_field_is_a_problem():
    f = FakeModel(responses=[call("c1", GOOD[:-1] + ',"hazmat":true}'),
                             call("c2", GOOD)])
    extract(f, Request())
    assert "hazmat" in f.calls[1].messages[-1].content
