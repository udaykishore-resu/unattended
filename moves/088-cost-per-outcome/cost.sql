WITH runs AS (
  SELECT c.run_id, c.workflow,
         SUM(c.in_tokens * p.in_rate
           + c.cache_read * p.read_rate
           + c.cache_write * p.write_rate
           + c.out_tokens * p.out_rate) / 1e6 AS cost
  FROM llm_calls c
  JOIN prices p ON p.model = c.model
  WHERE c.ts >= now() - interval '7 days'
  GROUP BY c.run_id, c.workflow
), latest AS (  -- one outcome per run: the newest
  SELECT DISTINCT ON (run_id) run_id, outcome
  FROM run_outcomes
  ORDER BY run_id, ts DESC
)
SELECT r.workflow, SUM(r.cost) AS total_cost,
       SUM(r.cost) / NULLIF(COUNT(*) FILTER
         (WHERE o.outcome = 'success'), 0)
         AS cost_per_win
FROM runs r
LEFT JOIN latest o USING (run_id)
GROUP BY r.workflow;
