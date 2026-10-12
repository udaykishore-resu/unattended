package hybrid

// The two searches behind Fuse. Both filter by
// tenant and expiry inside the query, before ORDER
// BY and LIMIT, so a filter can never run on an
// already-cut top 50. $1 is the tenant from the
// caller's session, never from model output.
// Move 47's expiry is the same predicate on both.

// VectorSQL is the pgvector side ($2 is the query
// embedding; <=> is cosine distance).
const VectorSQL = `SELECT id FROM chunks
WHERE tenant_id = $1
  AND (expires_at IS NULL OR expires_at >= now())
ORDER BY embedding <=> $2
LIMIT 50`

// KeywordSQL is the full-text side ($2 is the raw
// query text).
const KeywordSQL = `SELECT id FROM chunks
WHERE tenant_id = $1
  AND (expires_at IS NULL OR expires_at >= now())
  AND tsv @@ websearch_to_tsquery('english', $2)
ORDER BY ts_rank(tsv,
  websearch_to_tsquery('english', $2)) DESC
LIMIT 50`
