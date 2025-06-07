package engine

import (
	"os"
	"strings"
)

// SaveSQLDump exports all tables to a SQL file.
func SaveSQLDump(filename string) error {
	dbMu.RLock()
	defer dbMu.RUnlock()

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	for _, table := range Tables {
		if _, err := f.WriteString(buildCreateSQL(table)); err != nil {
			return err
		}
		for _, row := range table.Rows {
			if _, err := f.WriteString(buildInsertSQL(table, row)); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildCreateSQL(t *Table) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE ")
	b.WriteString(t.Name)
	b.WriteString(" (")
	for i, c := range t.Columns {
		b.WriteString(c.Name)
		b.WriteString(" ")
		b.WriteString(c.Type)
		if i != len(t.Columns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(");\n")
	return b.String()
}

func buildInsertSQL(t *Table, row []string) string {
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(t.Name)
	b.WriteString(" VALUES (")
	for i, val := range row {
		if t.Columns[i].Type == "INT" {
			b.WriteString(val)
		} else {
			b.WriteString("'")
			b.WriteString(strings.ReplaceAll(val, "'", "''"))
			b.WriteString("'")
		}
		if i != len(row)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(");\n")
	return b.String()
}
