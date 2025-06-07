package driver_test

import (
	"database/sql"
	"os"
	"testing"

	_ "minisql/driver"
	"minisql/engine"
)

func TestSQLDriver(t *testing.T) {
	// clean state
	_ = os.Remove("data.mdb")
	engine.Tables = make(map[string]*engine.Table)

	db, err := sql.Open("minidb", "")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if _, err := db.Exec("CREATE TABLE sqltest (id INT, name TEXT)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec("INSERT INTO sqltest VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	row := db.QueryRow("SELECT * FROM sqltest")
	var id, name string
	if err := row.Scan(&id, &name); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if id != "1" || name != "Alice" {
		t.Errorf("unexpected values: %s %s", id, name)
	}
}
