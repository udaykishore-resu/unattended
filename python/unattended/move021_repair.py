"""Move 21: validate, repair once, then reject.

The record arrives in a save_tender tool call (Move 20). Validate it in
code; on failure send one repair as that call's tool result; if the second
answer still fails, reject it for a person.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, replace
from typing import Optional

from .llm import Message, Model, Request, Response, ToolCall


class NoToolCallError(Exception):
    """The model answered in prose: send it to review."""


class RejectedError(Exception):
    """Failed after its one repair. Callers route it to a person."""


@dataclass
class Tender:
    shipper_ref: str
    origin_zip: Optional[str]
    dest_zip: Optional[str]
    pickup_date: str
    weight_lb: int


FIELDS = {"shipper_ref", "origin_zip", "dest_zip", "pickup_date", "weight_lb"}
ZIP5 = re.compile(r"^[0-9]{5}$")
ZIP9 = re.compile(r"^([0-9]{5})-[0-9]{4}$")


def one_call(r: Response, name: str) -> ToolCall:
    if len(r.tool_calls) != 1 or r.tool_calls[0].name != name:
        raise NoToolCallError("no tool call")
    return r.tool_calls[0]


def parse(raw: str) -> tuple[Optional[Tender], list[str]]:
    """Decode strictly and run every rule; one problem string per failure."""
    try:
        d = json.loads(raw)
    except json.JSONDecodeError as e:
        return None, [f"decode: {e}"]
    extra = set(d) - FIELDS
    if extra:
        return None, [f"decode: unknown field {sorted(extra)[0]!r}"]
    probs = []
    for name in ("origin_zip", "dest_zip"):
        z = d.get(name)
        if z is None:
            continue  # null is an answer, not a failure
        m = ZIP9.match(z)  # fix what code can fix before a call
        if m:
            d[name] = z = m.group(1)
        if not ZIP5.match(z):
            probs.append(f"{name}: {z!r} is not five digits")
    w = d.get("weight_lb")
    if not isinstance(w, int) or not 1 <= w <= 80000:
        probs.append(f"weight_lb: {w!r} is outside 1-80000")
    t = Tender(d.get("shipper_ref", ""), d.get("origin_zip"), d.get("dest_zip"),
               d.get("pickup_date", ""), w if isinstance(w, int) else 0)
    return t, probs


def repair_note(probs: list[str]) -> str:
    return ("These fields failed validation:\n- " + "\n- ".join(probs) +
            "\nCall save_tender again with fixed values. "
            "If a value is not in the email, use null.")


def extract(m: Model, req: Request) -> Tender:
    resp = m.complete(req)
    call = one_call(resp, "save_tender")
    t, probs = parse(call.input)
    if not probs:
        return t
    req = replace(req, messages=req.messages + [
        Message("assistant", resp.text, tool_calls=resp.tool_calls),
        Message("tool", repair_note(probs), tool_call_id=call.id),
    ])
    resp = m.complete(req)
    call = one_call(resp, "save_tender")
    t, probs = parse(call.input)
    if probs:
        raise RejectedError("; ".join(probs))
    return t
