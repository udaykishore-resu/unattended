"""Move 52: idempotency keys on every AI step.

The key is built from the business identity of the work (step name plus
what it acts on), length-prefixed, with no time, attempt number or prompt
version. `once` claims the key with a lease, runs the step, and stores the
result; a second caller gets the saved result.

Uses sqlite3 (stdlib) so the same ON CONFLICT logic as the Go version's
Postgres queries runs for real.
"""

from __future__ import annotations

import hashlib
import sqlite3
import time
from typing import Callable


class InFlightError(Exception):
    """Another worker holds a live lease."""


def key(step: str, *parts: str) -> str:
    b = "".join(f"{len(p)}:{p}|" for p in (step, *parts))
    return hashlib.sha256(b.encode()).hexdigest()[:32]


SCHEMA = """CREATE TABLE IF NOT EXISTS ai_steps (
  key TEXT PRIMARY KEY, lease_until REAL NOT NULL, output BLOB)"""

CLAIM = """INSERT INTO ai_steps (key, lease_until) VALUES (?, ?)
  ON CONFLICT (key) DO UPDATE SET lease_until = excluded.lease_until
  WHERE ai_steps.output IS NULL AND ai_steps.lease_until < ?"""


class Steps:
    def __init__(self, db: sqlite3.Connection, lease_s: float = 300.0):
        self.db, self.lease_s = db, lease_s
        db.execute(SCHEMA)

    def once(self, k: str, fn: Callable[[], bytes]) -> bytes:
        now = time.time()
        cur = self.db.execute(CLAIM, (k, now + self.lease_s, now))
        self.db.commit()
        if cur.rowcount != 1:
            row = self.db.execute(
                "SELECT output FROM ai_steps WHERE key = ?", (k,)).fetchone()
            if row[0] is None:
                raise InFlightError(k)
            return row[0]
        try:
            out = fn()
        except Exception:
            self.db.execute("UPDATE ai_steps SET lease_until = 0 "
                            "WHERE key = ? AND output IS NULL", (k,))
            self.db.commit()
            raise
        self.db.execute("UPDATE ai_steps SET output = ? WHERE key = ?", (out, k))
        self.db.commit()
        return out
