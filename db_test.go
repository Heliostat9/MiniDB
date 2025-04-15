package main

import (
	"minisql/engine"
	"strings"
	"testing"
)

func TestHandleCreateTable(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	resp, err := engine.HandleCommand("CREATE TABLE users (id, name)")
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
}

func TestHandleInsertAndSelect(t *testing.T) {
	engine.Tables = make(map[string]*engine.Table)

	_, _ = engine.HandleCommand("CREATE TABLE users (id, name)")

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
