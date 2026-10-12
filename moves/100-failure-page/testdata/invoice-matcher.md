# Invoice matcher: how it fails

STOP IT
Kill switch: flag `invoice_matcher` in automation_flags (UPDATE automation_flags SET enabled = false WHERE name = 'invoice_matcher')
Second way: kubectl scale deploy/invoice-matcher --replicas=0
Who may flip it: anyone on the platform on-call rota
Stopped when: invoice_matcher_runs_total stops rising for 5 min
Effect: new invoices go to the finance queue; items in flight finish
Safe to leave off for: 3 days (finance matches by hand)

WHAT NORMAL LOOKS LIKE
Runs per hour: 40   Review rate: 6%
Cost per day: 1.2M tokens   Dashboard: Invoice matcher

KNOWN FAILURE MODES
Review rate above 30% -> a carrier changed its invoice layout -> pause that carrier only
Cost per day doubles -> retry storm on provider 5xx -> check the retry wrapper's error rate

RECOVERY
Replay from: invoice_inbox table, rows with state = 'queued'
Idempotent: yes (key = invoice_id + prompt version)

OWNERS
Business: Marcus   Code: platform team
Escalate: Theo after 15 min, via the on-call pager
