// Package exceptionqueue holds Move 5's SQL: rank
// exception reasons by resolved count times median
// minutes to clear, with open items counted beside
// the ranking instead of hidden in it.
//
// Move 5: start from the exception queue, not the
// happy path.
//
// query.sql is the book's query. The test runs it in
// Postgres through psql (standard library only) and
// skips unless UNATTENDED_TEST_PSQL_URL holds a psql
// connection string, for example:
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test ./moves/005-exception-queue
//
// Tested on Postgres 16.
package exceptionqueue
