"""Move 52: idempotency keys on every AI step.

The key is built from the business identity of the work (step name plus
what it acts on), length-prefixed, with no time, attempt number or prompt
version. `once` claims the key with a lease, hands the key to the step (so
it can reach the downstream system), and stores the result; a second
caller gets the saved result.

If the step succeeds but storing its output fails, `once` raises
UnrecordedError and keeps the lease: nobody reruns the step until the
lease expires, and the takeover calls the step again with the same key.

Uses sqlite3 (stdlib) so the same ON CONFLICT logic as the Go version's
Postgres queries runs for real.
"""

from __future__ import annotations

import hashlib
import secrets
import sqlite3
import time
from typing import Callable


class InFlightError(Exception):
    """Another worker holds a live lease."""


class UnrecordedError(Exception):
    """The step ran but its output was not stored; the lease is kept."""

    def __init__(self, out: bytes, cause: Exception):
        super().__init__(f"step done but not recorded: {cause}")
        self.out = out


def key(step: str, *parts: str) -> str:
    b = "".join(f"{len(p)}:{p}|" for p in (step, *parts))
    return hashlib.sha256(b.encode()).hexdigest()[:32]


SCHEMA = """CREATE TABLE IF NOT EXISTS ai_steps (
  key TEXT PRIMARY KEY, lease_until REAL NOT NULL,
  owner TEXT NOT NULL, output BLOB)"""

CLAIM = """INSERT INTO ai_steps (key, lease_until, owner) VALUES (?, ?, ?)
  ON CONFLICT (key) DO UPDATE
  SET lease_until = excluded.lease_until, owner = excluded.owner
  WHERE ai_steps.output IS NULL AND ai_steps.lease_until < ?"""

RELEASE = """UPDATE ai_steps SET lease_until = 0
  WHERE key = ? AND owner = ? AND output IS NULL"""

FINISH = """UPDATE ai_steps SET output = COALESCE(output, ?)
  WHERE key = ? RETURNING output"""


class Steps:
    def __init__(self, db: sqlite3.Connection, lease_s: float = 300.0):
        self.db, self.lease_s = db, lease_s
        db.execute(SCHEMA)

    def _finish(self, k: str, out: bytes) -> bytes:
        row = self.db.execute(FINISH, (out, k)).fetchone()
        self.db.commit()
        if row is None:
            raise LookupError(k)
        return row[0]

    def once(self, k: str, fn: Callable[[str], bytes]) -> bytes:
        now, tok = time.time(), secrets.token_hex(16)
        cur = self.db.execute(CLAIM, (k, now + self.lease_s, tok, now))
        self.db.commit()
        if cur.rowcount != 1:
            row = self.db.execute(
                "SELECT output FROM ai_steps WHERE key = ?", (k,)).fetchone()
            if row[0] is None:
                raise InFlightError(k)
            return row[0]
        try:
            out = fn(k)  # the key goes downstream
        except Exception:
            self.db.execute(RELEASE, (k, tok))
            self.db.commit()
            raise
        try:
            return self._finish(k, out)
        except Exception as e:  # done but not recorded: keep the lease
            raise UnrecordedError(out, e) from e
