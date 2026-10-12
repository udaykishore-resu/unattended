-- Tables the query reads. run_id is Move 83's trace
-- ID: one per workflow run. Rates are per million
-- tokens and fictional.
CREATE TABLE prices (
  model text PRIMARY KEY,
  in_rate numeric, read_rate numeric,
  write_rate numeric, out_rate numeric);
CREATE TABLE llm_calls (
  run_id text, workflow text, model text, ts timestamptz,
  in_tokens int, cache_read int, cache_write int, out_tokens int);
CREATE TABLE run_outcomes (
  run_id text, outcome text, ts timestamptz);

INSERT INTO prices VALUES
  ('small', 1, 0.1, 1.25, 4),
  ('large', 10, 1, 12.5, 40);

-- r1: a retry (two calls) with cache traffic; succeeded.
-- r2: escalated to the large model; no outcome yet.
-- r3: outcome appended twice: failed, then succeeded.
-- r4: succeeded, then the label was changed: failed.
-- r5: older than the window; ignored.
INSERT INTO llm_calls VALUES
  ('r1','triage','small', now(), 1000, 0, 5000, 0),
  ('r1','triage','small', now(), 1000, 5000, 0, 100),
  ('r2','triage','small', now(), 2000, 0, 0, 50),
  ('r2','triage','large', now(), 2000, 0, 0, 50),
  ('r3','triage','small', now(), 1000, 0, 0, 0),
  ('r4','triage','small', now(), 1000, 0, 0, 0),
  ('r5','triage','large', now() - interval '30 days', 1000000, 0, 0, 0);
INSERT INTO run_outcomes VALUES
  ('r1','success', now()),
  ('r3','failure', now() - interval '2 hours'),
  ('r3','success', now() - interval '1 hour'),
  ('r4','success', now() - interval '2 hours'),
  ('r4','failure', now() - interval '1 hour');
