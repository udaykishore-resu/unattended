"""The tiny vendor-neutral model client (llm v2), ported from Go.

Same types as the book's front matter: Request, Message, Tool, ToolCall,
StopReason, Response and Usage, plus FakeModel for tests.
"""

from __future__ import annotations

import copy
import threading
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, Protocol


class StopReason(str, Enum):
    END = "end"
    MAX_TOKENS = "max_tokens"
    TOOL_USE = "tool_use"
    REFUSAL = "refusal"
    OTHER = "other"


@dataclass
class ToolCall:
    id: str
    name: str
    input: str = "{}"  # raw JSON, like json.RawMessage


@dataclass
class Tool:
    name: str
    description: str = ""
    input_schema: str = "{}"  # raw JSON Schema


@dataclass
class Message:
    """One turn. Assistant turns may carry tool_calls; a tool result is
    role "tool" with tool_call_id set to the call it answers."""

    role: str  # "user", "assistant", "tool"
    content: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    tool_call_id: str = ""


@dataclass
class Request:
    system: str = ""
    messages: list[Message] = field(default_factory=list)
    tools: list[Tool] = field(default_factory=list)
    max_tokens: int = 0
    schema: Optional[str] = None  # optional JSON Schema for output
    cache_system: bool = False  # mark system+tools as a cacheable prefix


@dataclass
class Usage:
    """Total input = input + cache_read + cache_write."""

    input_tokens: int = 0  # uncached input
    cache_read_tokens: int = 0
    cache_write_tokens: int = 0
    output_tokens: int = 0


@dataclass
class Response:
    text: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    usage: Usage = field(default_factory=Usage)
    stop_reason: Optional[StopReason] = None


class Model(Protocol):
    def complete(self, r: Request) -> Response: ...


class NoResponsesError(Exception):
    """fake: no responses"""


@dataclass
class FakeModel:
    """Replays canned responses in order and records a copy of every
    request it sees. Running out of responses is an error on purpose."""

    responses: list[Response] = field(default_factory=list)
    err: Optional[Exception] = None
    calls: list[Request] = field(default_factory=list)
    _mu: threading.Lock = field(default_factory=threading.Lock, repr=False)

    def complete(self, r: Request) -> Response:
        with self._mu:
            # Copy: an agent loop appends to messages.
            rec = copy.copy(r)
            rec.messages = list(r.messages)
            self.calls.append(rec)
            if self.err is not None:
                raise self.err
            if not self.responses:
                raise NoResponsesError("fake: no responses")
            resp = copy.copy(self.responses.pop(0))
            if resp.stop_reason is None:
                resp.stop_reason = StopReason.END
                if resp.tool_calls:
                    resp.stop_reason = StopReason.TOOL_USE
            return resp
