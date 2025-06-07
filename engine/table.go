package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Column struct {
	Name string
	Type string
}

type Table struct {
	Name    string
	Columns []Column
	Rows    [][]string
}

var Tables = make(map[string]*Table)

func Init() error {
	return LoadBinaryDB()
}

func HandleCommand(query string) (string, error) {
	query = strings.TrimSpace(query)
	queryUpper := strings.ToUpper(query)

	switch {
	case strings.HasPrefix(queryUpper, "CREATE TABLE"):
		return handleCreateTable(query)
	case strings.HasPrefix(queryUpper, "INSERT INTO"):
		return handleInsert(query)
	case strings.HasPrefix(queryUpper, "UPDATE"):
		return handleUpdate(query)
	case strings.HasPrefix(queryUpper, "SELECT"):
		return handleSelect(query)
	case strings.HasPrefix(queryUpper, "DUMP"):
		return handleDump(query)
	default:
		return "", errors.New("unsupported command")
	}
}

func handleCreateTable(query string) (string, error) {
	open := strings.Index(query, "(")
	close := strings.Index(query, ")")

	if open == -1 || close == -1 || open > close {
		return "", errors.New("invalid syntax for CREATE TABLE")
	}

	header := strings.TrimSpace(query[12:open])
	cols := strings.Split(query[open+1:close], ",")

	var columns []Column
	for _, col := range cols {
		parts := strings.Fields(strings.TrimSpace(col))
		if len(parts) == 0 {
			continue
		}
		name := parts[0]
		colType := "TEXT"
		if len(parts) > 1 {
			colType = strings.ToUpper(parts[1])
		}
		columns = append(columns, Column{Name: name, Type: colType})
	}

	table := &Table{
		Name:    header,
		Columns: columns,
		Rows:    [][]string{},
	}

	dbMu.Lock()
	Tables[header] = table
	dbMu.Unlock()

	returnMsg := fmt.Sprintf("Table '%s' created.", header)

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}
	return returnMsg, nil
}

func handleInsert(query string) (string, error) {
	queryUpper := strings.ToUpper(query)
	valuesIdx := strings.Index(queryUpper, "VALUES")
	if valuesIdx == -1 {
		return "", errors.New("invalid syntax for INSERT")
	}

	intoPart := strings.Fields(query[:valuesIdx])
	if len(intoPart) < 3 {
		return "", errors.New("invalid INSERT INTO syntax")
	}
	tableName := intoPart[2]

	valuesRaw := strings.TrimSpace(query[valuesIdx+len("VALUES"):])
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

	dbMu.Lock()
	table, exists := Tables[tableName]
	if !exists {
		dbMu.Unlock()
		return "", errors.New("table does not exist")
	}

	if len(row) != len(table.Columns) {
		dbMu.Unlock()
		return "", errors.New("columns count does not match")
	}

	for i, val := range row {
		if table.Columns[i].Type == "INT" {
			if _, err := strconv.Atoi(val); err != nil {
				dbMu.Unlock()
				return "", fmt.Errorf("invalid INT value for column %s", table.Columns[i].Name)
			}
		}
	}

	table.Rows = append(table.Rows, row)
	dbMu.Unlock()

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}
	return "1 row inserted.", nil
}

func handleSelect(query string) (string, error) {
	tokens := strings.Fields(query)
	if len(tokens) < 4 || tokens[1] != "*" || tokens[2] != "FROM" {
		return "", errors.New("only SELECT * FROM <table> supported for now")
	}

	tableName := tokens[3]
	dbMu.RLock()
	table, exists := Tables[tableName]
	if !exists {
		dbMu.RUnlock()
		return "", errors.New("table does not exist")
	}

	var builder strings.Builder
	colNames := make([]string, len(table.Columns))
	for i, c := range table.Columns {
		colNames[i] = c.Name
	}
	builder.WriteString(strings.Join(colNames, "\t") + "\n")

	for _, row := range table.Rows {
		builder.WriteString(strings.Join(row, "\t") + "\n")
	}
	dbMu.RUnlock()

	return builder.String(), nil
}

func handleUpdate(query string) (string, error) {
	queryUpper := strings.ToUpper(query)
	setIdx := strings.Index(queryUpper, " SET ")
	if setIdx == -1 {
		return "", errors.New("invalid syntax for UPDATE")
	}

	whereIdx := strings.Index(queryUpper, " WHERE ")
	if whereIdx == -1 {
		return "", errors.New("UPDATE without WHERE is not supported")
	}

	tableName := strings.TrimSpace(query[6:setIdx])

	dbMu.Lock()
	table, exists := Tables[tableName]
	if !exists {
		dbMu.Unlock()
		return "", errors.New("table does not exist")
	}

	assignmentsRaw := query[setIdx+5 : whereIdx]
	condRaw := query[whereIdx+7:]

	// Parse condition
	condParts := strings.SplitN(condRaw, "=", 2)
	if len(condParts) != 2 {
		dbMu.Unlock()
		return "", errors.New("invalid WHERE syntax")
	}
	condCol := strings.TrimSpace(condParts[0])
	condVal := strings.Trim(strings.TrimSpace(condParts[1]), "'")

	condIdx := -1
	for i, c := range table.Columns {
		if c.Name == condCol {
			condIdx = i
			break
		}
	}
	if condIdx == -1 {
		dbMu.Unlock()
		return "", fmt.Errorf("unknown column %s", condCol)
	}

	// Parse assignments
	assignmentList := strings.Split(assignmentsRaw, ",")
	updates := make(map[int]string)
	for _, a := range assignmentList {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) != 2 {
			dbMu.Unlock()
			return "", errors.New("invalid SET syntax")
		}
		col := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "'")

		idx := -1
		for i, c := range table.Columns {
			if c.Name == col {
				idx = i
				break
			}
		}
		if idx == -1 {
			dbMu.Unlock()
			return "", fmt.Errorf("unknown column %s", col)
		}
		if table.Columns[idx].Type == "INT" {
			if _, err := strconv.Atoi(val); err != nil {
				dbMu.Unlock()
				return "", fmt.Errorf("invalid INT value for column %s", col)
			}
		}
		updates[idx] = val
	}

	updated := 0
	for i, row := range table.Rows {
		if row[condIdx] == condVal {
			for idx, val := range updates {
				if idx < len(row) {
					row[idx] = val
				}
			}
			table.Rows[i] = row
			updated++
		}
	}
	dbMu.Unlock()

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d rows updated.", updated), nil
}

func handleDump(query string) (string, error) {
	tokens := strings.Fields(query)
	filename := "dump.sql"
	if len(tokens) > 1 {
		filename = tokens[1]
	}
	if err := SaveSQLDump(filename); err != nil {
		return "", err
	}
	return fmt.Sprintf("Dump saved to %s.", filename), nil
}
