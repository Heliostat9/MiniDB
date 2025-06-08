// Package driver provides a database/sql driver for MiniDB.
// Import for side effects via:
//
//	_ "minisql/driver"
package driver

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"

	"minisql/engine"
)

// Driver implements database/sql/driver.Driver for MiniDB.
// It registers itself under the name "minidb" when imported.
type Driver struct{}

func init() { sql.Register("minidb", &Driver{}) }

// Open initializes the database and returns a new connection.
func (d *Driver) Open(name string) (driver.Conn, error) {
	if err := engine.Init(); err != nil {
		return nil, err
	}
	return &conn{}, nil
}

type conn struct{}

func (c *conn) Prepare(query string) (driver.Stmt, error) { return &stmt{query: query}, nil }
func (c *conn) Close() error                              { return nil }
func (c *conn) Begin() (driver.Tx, error) {
	tx := engine.BeginTx()
	return &sqlTx{tx: tx}, nil
}

type sqlTx struct{ tx *engine.Tx }

func (t *sqlTx) Commit() error   { return t.tx.Commit() }
func (t *sqlTx) Rollback() error { t.tx.Rollback(); return nil }

type stmt struct{ query string }

func (s *stmt) Close() error { return nil }

// NumInput returns -1 since MiniDB does not support placeholders.
func (s *stmt) NumInput() int { return -1 }

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	if _, err := engine.Execute(s.query); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	res, err := engine.Execute(s.query)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(res), "\n")
	if len(lines) == 0 {
		return &rows{}, nil
	}
	cols := strings.Split(lines[0], "\t")
	data := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		data = append(data, strings.Split(line, "\t"))
	}
	return &rows{columns: cols, rows: data}, nil
}

type rows struct {
	columns []string
	rows    [][]string
	idx     int
}

func (r *rows) Columns() []string { return r.columns }
func (r *rows) Close() error      { return nil }

func (r *rows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.idx]
	r.idx++
	for i, v := range row {
		dest[i] = v
	}
	return nil
}
