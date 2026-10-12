import threading

import pytest

from unattended.llm import Request, Response, StopReason, Usage
from unattended.move089_budget import (Budgeted, BudgetError, Ledger,
                                       Permanent, retry, total)


class Flaky:
    """Fails its first `fails` calls with no usage, then answers."""

    def __init__(self, fails=0, usage=Usage()):
        self.fails, self.usage, self.calls = fails, usage, 0
        self._mu = threading.Lock()

    def complete(self, r):
        with self._mu:
            self.calls += 1
            if self.calls <= self.fails:
                raise TimeoutError("timeout")
        return Response(text="ok", usage=self.usage, stop_reason=StopReason.END)


WORST = 100 + 500  # est_in + max_tokens


def budgeted(model, ledger):
    return Budgeted(model, ledger, key=lambda: "triage/acme", est_in=lambda r: 100)


def test_budget_charges_every_retry():
    ledger = Ledger({"triage/acme": 100_000})
    f = Flaky(fails=2, usage=Usage(40, 300, 20, 60))
    m = budgeted(f, ledger)
    retry(lambda: m.complete(Request(max_tokens=500)), tries=5, base=0.001, ceil=0.002)
    assert f.calls == 3
    assert ledger.used["triage/acme"] == 2 * WORST + (40 + 300 + 20 + 60)


def test_runaway_stops_and_refusal_is_not_retried():
    ledger = Ledger({"triage/acme": 3 * WORST})
    f = Flaky(usage=Usage(input_tokens=WORST))
    m = budgeted(f, ledger)
    ok = 0
    for _ in range(10):
        try:
            m.complete(Request(max_tokens=500))
            ok += 1
        except BudgetError:
            pass
    assert ok == 3 and f.calls == 3

    tries = 0

    def once():
        nonlocal tries
        tries += 1
        try:
            return m.complete(Request(max_tokens=500))
        except BudgetError as e:
            raise Permanent(str(e)) from e

    with pytest.raises(Permanent):
        retry(once, tries=5, base=0.001, ceil=0.001)
    assert tries == 1


def test_no_budget_no_run():
    with pytest.raises(BudgetError):
        budgeted(Flaky(), Ledger({})).complete(Request(max_tokens=10))


def test_concurrent_reservations_stay_inside_the_limit():
    ledger = Ledger({"triage/acme": 10 * WORST})
    f = Flaky(usage=Usage(input_tokens=WORST))
    m = budgeted(f, ledger)

    def go():
        try:
            m.complete(Request(max_tokens=500))
        except BudgetError:
            pass

    ts = [threading.Thread(target=go) for _ in range(50)]
    for t in ts:
        t.start()
    for t in ts:
        t.join()
    assert f.calls == 10 and ledger.used["triage/acme"] == 10 * WORST


def test_total_counts_cache():
    assert total(Usage(1, 2, 3, 4)) == 10


def test_zero_usage_on_success_keeps_worst_case():
    ledger = Ledger({"triage/acme": 100_000})
    budgeted(Flaky(), ledger).complete(Request(max_tokens=500))
    assert ledger.used["triage/acme"] == WORST


def test_refusal_is_permanent_without_rewrapping():
    m = budgeted(Flaky(), Ledger({"triage/acme": 1}))
    tries = 0

    def once():
        nonlocal tries
        tries += 1
        return m.complete(Request(max_tokens=500))

    with pytest.raises(BudgetError):
        retry(once, tries=3, base=0.001, ceil=0.001)
    assert tries == 1
