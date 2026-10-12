import sqlite3
import time

import pytest

from unattended.move052_idempotency import (InFlightError, Steps,
                                            UnrecordedError, key)


def test_key_stable_across_prompt_versions():
    st = Steps(sqlite3.connect(":memory:"))
    sends = []

    def step(prompt_ver):
        def fn(k):
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


def test_step_receives_the_key():
    st = Steps(sqlite3.connect(":memory:"))
    seen = []
    k = key("file-ticket", "event:e-1")
    st.once(k, lambda kk: seen.append(kk) or b"T-1")
    assert seen == [k]


def test_failure_releases_the_lease():
    st = Steps(sqlite3.connect(":memory:"))
    k = key("summarize", "event:e-9")

    def boom(_):
        raise TimeoutError("model timed out")

    with pytest.raises(TimeoutError):
        st.once(k, boom)
    assert st.once(k, lambda _: b"ok") == b"ok"


def test_live_lease_is_in_flight_and_expired_lease_is_taken_over():
    db = sqlite3.connect(":memory:")
    st = Steps(db)
    k = key("x", "y")
    db.execute("INSERT INTO ai_steps (key, lease_until, owner) VALUES (?, ?, ?)",
               (k, time.time() + 3600, "other"))
    with pytest.raises(InFlightError):
        st.once(k, lambda _: b"never")
    db.execute("UPDATE ai_steps SET lease_until = 0 WHERE key = ?", (k,))
    assert st.once(k, lambda _: b"took over") == b"took over"


def test_done_but_unrecorded_keeps_the_lease():
    db = sqlite3.connect(":memory:")
    st = Steps(db)
    k = key("file-ticket", "event:e-77")
    created = {}

    def step(kk):  # a ticket API that honors the key
        created.setdefault(kk, b"T-77")
        return created[kk]

    real_finish = st._finish
    st._finish = lambda *_: (_ for _ in ()).throw(sqlite3.OperationalError("reset"))
    with pytest.raises(UnrecordedError) as e:
        st.once(k, step)
    assert e.value.out == b"T-77"
    st._finish = real_finish
    with pytest.raises(InFlightError):  # no rerun inside the lease
        st.once(k, step)
    db.execute("UPDATE ai_steps SET lease_until = 0 WHERE key = ?", (k,))
    assert st.once(k, step) == b"T-77" and len(created) == 1


def test_stale_holder_cannot_release_and_first_finish_wins():
    db = sqlite3.connect(":memory:")
    st = Steps(db)
    k = key("draft", "invoice:9")

    def a_step(_):
        # A's lease expires mid-step and B takes over and finishes.
        db.execute("UPDATE ai_steps SET lease_until = 0 WHERE key = ?", (k,))
        assert st.once(k, lambda _: b"B") == b"B"
        return b"A"

    assert st.once(k, a_step) == b"B"  # A gets the stored output

    k2 = key("draft", "invoice:10")

    def a_fails(_):
        db.execute("UPDATE ai_steps SET lease_until = 0 WHERE key = ?", (k2,))
        db.execute(  # B claims and is still running
            "UPDATE ai_steps SET lease_until = ?, owner = 'B' WHERE key = ?",
            (time.time() + 3600, k2))
        raise TimeoutError("A timed out")

    with pytest.raises(TimeoutError):
        st.once(k2, a_fails)
    with pytest.raises(InFlightError):  # A did not release B's lease
        st.once(k2, lambda _: b"C")
