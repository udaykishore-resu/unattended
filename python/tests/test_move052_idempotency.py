import sqlite3
import time

import pytest

from unattended.move052_idempotency import InFlightError, Steps, key


def test_key_stable_across_prompt_versions():
    st = Steps(sqlite3.connect(":memory:"))
    sends = []

    def step(prompt_ver):
        def fn():
            sends.append(prompt_ver)  # the side effect: an email
            return f"sent with {prompt_ver}".encode()
        return fn

    k = key("dispute-email", "carrier:RDX", "invoice:40117")
    first = st.once(k, step("v7"))
    # A prompt fix (v7 -> v8) leaves the business identity, so the key, alone.
    assert key("dispute-email", "carrier:RDX", "invoice:40117") == k
    second = st.once(k, step("v8"))
    assert sends == ["v7"] and first == second == b"sent with v7"


def test_length_prefix():
    assert key("s", "ab", "c") != key("s", "a", "bc")


def test_failure_releases_the_lease():
    st = Steps(sqlite3.connect(":memory:"))
    k = key("summarize", "event:e-9")

    def boom():
        raise TimeoutError("model timed out")

    with pytest.raises(TimeoutError):
        st.once(k, boom)
    assert st.once(k, lambda: b"ok") == b"ok"


def test_live_lease_is_in_flight_and_expired_lease_is_taken_over():
    db = sqlite3.connect(":memory:")
    st = Steps(db)
    k = key("x", "y")
    db.execute("INSERT INTO ai_steps (key, lease_until) VALUES (?, ?)",
               (k, time.time() + 3600))
    with pytest.raises(InFlightError):
        st.once(k, lambda: b"never")
    db.execute("UPDATE ai_steps SET lease_until = 0 WHERE key = ?", (k,))
    assert st.once(k, lambda: b"took over") == b"took over"
