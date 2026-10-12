// Package cost holds Move 88's query, cost.sql, and a
// test that runs it against Postgres with psql.
//
//	UNATTENDED_TEST_PSQL_URL=postgres://... \
//	    go test ./moves/088-cost-per-outcome
//
// Without the variable the test skips.
//
// Move 88: cost per successful outcome, not per call.
package cost
