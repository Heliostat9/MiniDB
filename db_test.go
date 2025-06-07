package main

import (
	"minisql/engine"
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
