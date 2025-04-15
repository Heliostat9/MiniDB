package engine

import (
	"errors"
	"fmt"
	"strings"
)

type Table struct {
	Name    string
	Columns []string
	Rows    [][]string
}

var tables = make(map[string]*Table)

const dbFile = "data.db"

func Init() error {
	return LoadBinaryDB()
}

func handleCreateTable(query string) (string, error) {
	open := strings.Index(query, "(")
	close := strings.Index(query, ")")

	if open == -1 || close == -1 || open > close {
		return "", errors.New("invalid syntex for CREATE TABLE")
	}

	header := strings.TrimSpace(query[12:open])
	cols := strings.Split(query[open+1:close], ",")

	var columnsNames []string
	for _, col := range cols {
		columnsNames = append(columnsNames, strings.TrimSpace(col))
	}

	table := &Table{
		Name:    header,
		Columns: columnsNames,
		Rows:    [][]string{},
	}

	tables[header] = table
	returnMsg := fmt.Sprintf("Table '%s' created.", header)

	return returnMsg, SaveBinaryDB()
}

func handleInsert(query string) (string, error) {
	parts := strings.SplitN(query, "VALUES", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid syntax for INSERT")
	}

	intoPart := strings.Fields(parts[0])
	if len(intoPart) < 3 {
		return "", errors.New("invalid INSERT INTO syntax")
	}
	tableName := intoPart[2]
	table, exists := tables[tableName]
	if !exists {
		return "", errors.New("table does not exist")
	}

	valuesRaw := strings.TrimSpace(parts[1])
	open := strings.Index(valuesRaw, "(")
	close := strings.Index(valuesRaw, ")")
	if open == -1 || close == -1 || open > close {
		return "", errors.New("invalid VALUES syntax")
	}

	vals := strings.Split(valuesRaw[open+1:close], ",")
	var row []string
	for _, v := range vals {
		row = append(row, strings.Trim(strings.TrimSpace(v), "'"))
	}

	if len(row) != len(table.Columns) {
		return "", errors.New("columns count does not match")
	}

	table.Rows = append(table.Rows, row)
	return "1 row iserted.", SaveBinaryDB()
}

func handleSelect(query string) (string, error) {
	tokens := strings.Fields(query)
	if len(tokens) < 4 || tokens[1] != "*" || tokens[2] != "FROM" {
		return "", errors.New("only SELECT * FROM <table> supported for now")
	}

	tableName := tokens[3]
	table, exists := tables[tableName]

	if !exists {
		return "", errors.New("table does not exist")
	}

	var builder strings.Builder
	builder.WriteString(strings.Join(table.Columns, "\t") + "\n")

	for _, row := range table.Rows {
		builder.WriteString(strings.Join(row, "\t") + "\n")
	}

	return builder.String(), nil
}
