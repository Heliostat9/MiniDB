package main

import (
	"minisql/engine"
	"os"
	"strings"
	"testing"
)

func TestHandleCreateTable(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	resp, err := engine.HandleCommand("CREATE TABLE users (id INT, name TEXT)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(resp, "created") {
		t.Errorf("unexpected response: %s", resp)
	}

	table := engine.Tables["users"]
	if table == nil || len(table.Columns) != 2 {
		t.Errorf("table creation failed: %+v", table)
	}
	if table.Columns[0].Type != "INT" {
		t.Errorf("expected INT column type, got %s", table.Columns[0].Type)
	}
}

func TestHandleInsertAndSelect(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	_, _ = engine.HandleCommand("CREATE TABLE users (id INT, name TEXT)")

	resp, err := engine.HandleCommand("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("insert failed: %v, resp: %s", err, resp)
	}

	result, err := engine.HandleCommand("SELECT * FROM users")
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	if !strings.Contains(result, "Alice") {
		t.Errorf("select returned wrong data: %s", result)
	}
}

func TestHandleUpdate(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	_, _ = engine.HandleCommand("CREATE TABLE users (id INT, name TEXT)")
	_, _ = engine.HandleCommand("INSERT INTO users VALUES (1, 'Alice')")

	resp, err := engine.HandleCommand("UPDATE users SET name='Bob' WHERE id=1")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(resp, "1 rows updated") {
		t.Errorf("unexpected update response: %s", resp)
	}

	result, err := engine.HandleCommand("SELECT * FROM users")
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}
	if !strings.Contains(result, "Bob") {
		t.Errorf("update did not apply: %s", result)
	}
}

func TestSaveSQLDump(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	_, _ = engine.HandleCommand("CREATE TABLE users (id INT, name TEXT)")
	_, _ = engine.HandleCommand("INSERT INTO users VALUES (1, 'Alice')")

	if err := engine.SaveSQLDump("test_dump.sql"); err != nil {
		t.Fatalf("dump failed: %v", err)
	}

	data, err := os.ReadFile("test_dump.sql")
	if err != nil {
		t.Fatalf("read dump failed: %v", err)
	}
	_ = os.Remove("test_dump.sql")

	dump := string(data)
	if !strings.Contains(dump, "CREATE TABLE users") ||
		!strings.Contains(dump, "INSERT INTO users VALUES (1, 'Alice')") {
		t.Errorf("dump content incorrect: %s", dump)
	}
}

func TestAdditionalTypes(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	_, err := engine.HandleCommand("CREATE TABLE metrics (score FLOAT, active BOOL)")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = engine.HandleCommand("INSERT INTO metrics VALUES (3.14, true)")
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	result, err := engine.HandleCommand("SELECT * FROM metrics")
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	if !strings.Contains(result, "3.14") || !strings.Contains(result, "true") {
		t.Errorf("select returned wrong data: %s", result)
	}
}

func TestWALRecovery(t *testing.T) {
	_ = os.Remove("data.mdb")
	_ = os.Remove("data.wal")
	wal := "CREATE TABLE waltest (id INT)\nINSERT INTO waltest VALUES (1)\n"
	if err := os.WriteFile("data.wal", []byte(wal), 0600); err != nil {
		t.Fatalf("write wal: %v", err)
	}
	if err := engine.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	table := engine.Tables["waltest"]
	if table == nil || len(table.Rows) != 1 {
		t.Fatalf("wal replay failed: %+v", table)
	}
	if _, err := os.Stat("data.wal"); !os.IsNotExist(err) {
		t.Errorf("wal file not cleared")
	}
}
