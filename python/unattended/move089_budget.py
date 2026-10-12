"""Move 89: token budgets per workflow and per tenant.

Reserve the worst case before every call, settle the real number after.
The wrapper sits inside the retry layer, so every attempt is charged.
"""

from __future__ import annotations

import random
import threading
import time
from dataclasses import dataclass
from typing import Callable

from .llm import Model, Request, Response, Usage


class Permanent(Exception):
    """Move 56: wraps errors no retry can fix."""


class BudgetError(Permanent):
    """The key has no limit, or the worst case won't fit.

    A Permanent by construction, so the retry layer stops at once.
    """


def total(u: Usage) -> int:
    """All four fields: cache writes and reads are spend too."""
    return u.input_tokens + u.cache_read_tokens + u.cache_write_tokens + u.output_tokens


class Ledger:
    def __init__(self, limits: dict[str, int]):
        self._mu = threading.Lock()
        self.used: dict[str, int] = {}
        self.limit = dict(limits)

    def reserve(self, k: str, n: int) -> None:
        with self._mu:
            lim = self.limit.get(k)
            if lim is None or self.used.get(k, 0) + n > lim:
                raise BudgetError(k)
            self.used[k] = self.used.get(k, 0) + n

    def settle(self, k: str, reserved: int, got: int) -> None:
        with self._mu:
            self.used[k] = self.used.get(k, 0) + got - reserved


def est_bytes(r: Request) -> int:
    n = len(r.system) + sum(len(m.content) for m in r.messages)
    return n // 4 + 1


@dataclass
class Budgeted:
    next: Model
    ledger: Ledger
    key: Callable[[], str]  # workflow and tenant
    est_in: Callable[[Request], int] = est_bytes

    def complete(self, r: Request) -> Response:
        k = self.key()
        worst = self.est_in(r) + r.max_tokens
        self.ledger.reserve(k, worst)
        got = worst  # exception or no usage: assume spent
        try:
            resp = self.next.complete(r)
            n = total(resp.usage)
            if n > 0:  # zero is "unknown", not "free"
                got = n
            return resp
        except Exception as e:
            n = total(getattr(e, "usage", Usage()))
            if n > 0:
                got = n
            raise
        finally:
            self.ledger.settle(k, worst, got)


def retry(fn: Callable[[], object], tries: int, base: float, ceil: float):
    """Move 56: exponential backoff with full jitter; Permanent stops."""
    err: Exception | None = None
    for i in range(tries):
        try:
            return fn()
        except Permanent:
            raise
        except Exception as e:  # noqa: BLE001 - retry anything else
            err = e
            time.sleep(random.uniform(0, min(ceil, base * (2 ** i))))
    assert err is not None
    raise err
