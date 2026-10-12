-- Postgres. created_at is timestamptz;
-- minutes_to_clear is numeric, NULL while open.
-- The subquery is there because ORDER BY can't
-- do math on output aliases.
SELECT reason, n_done, n_open, p50_min
FROM (
  SELECT reason,
         count(minutes_to_clear) AS n_done,
         count(*) - count(minutes_to_clear) AS n_open,
         percentile_cont(0.5) WITHIN GROUP
           (ORDER BY minutes_to_clear) AS p50_min
  FROM invoice_exceptions
  WHERE created_at > now() - interval '30 days'
  GROUP BY reason
) by_reason
ORDER BY n_done * p50_min DESC NULLS LAST;
