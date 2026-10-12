// Package sqlfake is a tiny database/sql driver for
// tests. It is not part of the book: it lets the
// moves that take a *sql.DB (Moves 7, 40, 51, 52 and
// 53) run their real code with no database server.
//
// A Handler sees every query with its arguments and
// returns the columns and rows to hand back. Exec
// reports len(rows) as the rows affected.
package sqlfake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
)

// Handler answers one query.
type Handler func(query string, args []any) (
	cols []string, rows [][]any, err error)

// Open returns a *sql.DB whose every query goes to h.
func Open(h Handler) *sql.DB {
	return sql.OpenDB(connector{h})
}

type connector struct{ h Handler }

func (c connector) Connect(context.Context) (driver.Conn, error) {
	return conn(c), nil
}

func (c connector) Driver() driver.Driver { return drv{} }

type drv struct{}

func (drv) Open(string) (driver.Conn, error) {
	return nil, errors.New("sqlfake: use Open")
}

type conn struct{ h Handler }

func (c conn) Prepare(q string) (driver.Stmt, error) {
	return stmt{c.h, q}, nil
}
func (conn) Close() error { return nil }
func (conn) Begin() (driver.Tx, error) {
	return nil, errors.New("sqlfake: no transactions")
}

func values(nv []driver.NamedValue) []any {
	out := make([]any, len(nv))
	for i, v := range nv {
		out[i] = v.Value
	}
	return out
}

func (c conn) QueryContext(_ context.Context, q string,
	nv []driver.NamedValue) (driver.Rows, error) {
	cols, rs, err := c.h(q, values(nv))
	if err != nil {
		return nil, err
	}
	return &rows{cols: cols, data: rs}, nil
}

func (c conn) ExecContext(_ context.Context, q string,
	nv []driver.NamedValue) (driver.Result, error) {
	_, rs, err := c.h(q, values(nv))
	if err != nil {
		return nil, err
	}
	return driver.RowsAffected(len(rs)), nil
}

type stmt struct {
	h Handler
	q string
}

func (stmt) Close() error  { return nil }
func (stmt) NumInput() int { return -1 }
func (s stmt) Exec(args []driver.Value) (driver.Result, error) {
	return conn{s.h}.ExecContext(context.Background(), s.q, named(args))
}
func (s stmt) Query(args []driver.Value) (driver.Rows, error) {
	return conn{s.h}.QueryContext(context.Background(), s.q, named(args))
}

func named(args []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(args))
	for i, a := range args {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: a}
	}
	return out
}

type rows struct {
	cols []string
	data [][]any
	i    int
}

func (r *rows) Columns() []string { return r.cols }
func (r *rows) Close() error      { return nil }
func (r *rows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	for j, v := range r.data[r.i] {
		dest[j] = v
	}
	r.i++
	return nil
}
