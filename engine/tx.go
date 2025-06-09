package engine

import (
	"os"
)

// Tx represents a simple transaction holding pending WAL entries
// and a snapshot of tables for rollback.
type Tx struct {
	wal      []string
	snapshot map[string]*Table
}

var txCtx *Tx

// BeginTx starts a new transaction and locks the database for exclusive access.
func BeginTx() *Tx {
	dbMu.Lock()
	txCtx = &Tx{snapshot: cloneTables(Tables)}
	return txCtx
}

// Exec executes a query within the transaction using the normal command handler.
func (tx *Tx) Exec(query string) (string, error) { return HandleCommand(query) }

// Commit writes all pending WAL entries and persists the DB to disk.
func (tx *Tx) Commit() error {
	// write accumulated WAL entries
	walMu.Lock()
	f, err := os.OpenFile(walFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		walMu.Unlock()
		return err
	}
	for _, line := range tx.wal {
		if _, err := f.WriteString(line + "\n"); err != nil {
			_ = f.Close()
			walMu.Unlock()
			return err
		}
	}
	_ = f.Close()
	walMu.Unlock()

	txCtx = nil
	if err := saveBinaryDBNoLock(); err != nil {
		dbMu.Unlock()
		return err
	}
	if err := clearWAL(); err != nil {
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return nil
}

// Rollback restores the snapshot state and discards changes.
func (tx *Tx) Rollback() {
	Tables = tx.snapshot
	txCtx = nil
	dbMu.Unlock()
}

// cloneTables performs a deep copy of tables for transaction snapshots.
func cloneTables(src map[string]*Table) map[string]*Table {
	newMap := make(map[string]*Table, len(src))
	for name, tbl := range src {
		t := &Table{
			Name:    tbl.Name,
			Columns: append([]Column(nil), tbl.Columns...),
			Rows:    make([]Row, len(tbl.Rows)),
		}
		for i, row := range tbl.Rows {
			nr := make(Row, len(row))
			copy(nr, row)
			t.Rows[i] = nr
		}
		if len(tbl.Indexes) > 0 {
			t.Indexes = make(map[string]*Index, len(tbl.Indexes))
			for col, idx := range tbl.Indexes {
				ni := &Index{Column: idx.Column, idx: idx.idx, Values: make(map[interface{}][]int)}
				for v, arr := range idx.Values {
					ni.Values[v] = append([]int(nil), arr...)
				}
				t.Indexes[col] = ni
			}
		}
		newMap[name] = t
	}
	return newMap
}
