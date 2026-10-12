"""Move 41: loop detection: same tool, same args, twice.

The second identical call gets a ToolError the model can act on; the third
ends the run. "Same arguments" ignores key order: decode and re-encode with
sorted keys.
"""

from __future__ import annotations

import json
from collections import Counter

from .llm import ToolCall
from .move033_gate import ToolError


class LoopError(Exception):
    """loop: third identical call"""


class LoopGuard:
    def __init__(self) -> None:
        self.seen: Counter[str] = Counter()

    def check(self, c: ToolCall) -> None:
        try:
            v = json.loads(c.input)
        except json.JSONDecodeError:
            v = c.input
        k = c.name + "\x00" + json.dumps(v, sort_keys=True, separators=(",", ":"))
        self.seen[k] += 1
        n = self.seen[k]
        if n == 1:
            return
        if n == 2:
            raise ToolError("repeat_call", "you already made this exact call",
                            "use the earlier result or stop")
        raise LoopError("loop: third identical call")
