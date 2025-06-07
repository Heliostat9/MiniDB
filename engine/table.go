package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Column struct {
	Name string
	Type string
}

type Row []interface{}

type Table struct {
	Name    string
	Columns []Column
	Rows    []Row
	mu      sync.RWMutex
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
		Rows:    []Row{},
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

	dbMu.RLock()
	table, exists := Tables[tableName]
	dbMu.RUnlock()
	if !exists {
		return "", errors.New("table does not exist")
	}

	var row Row
	for i, v := range vals {
		val := strings.Trim(strings.TrimSpace(v), "'")
		if i < len(table.Columns) && table.Columns[i].Type == "INT" {
			num, err := strconv.Atoi(val)
			if err != nil {
				return "", fmt.Errorf("invalid INT value for column %s", table.Columns[i].Name)
			}
			row = append(row, num)
		} else {
			row = append(row, val)
		}
	}

	if len(row) != len(table.Columns) {
		return "", errors.New("columns count does not match")
	}

	table.mu.Lock()
	table.Rows = append(table.Rows, row)
	table.mu.Unlock()

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

	table.mu.RLock()
	for _, row := range table.Rows {
		strVals := make([]string, len(row))
		for i, v := range row {
			strVals[i] = fmt.Sprint(v)
		}
		builder.WriteString(strings.Join(strVals, "\t") + "\n")
	}
	table.mu.RUnlock()
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

	dbMu.RLock()
	table, exists := Tables[tableName]
	dbMu.RUnlock()
	if !exists {
		return "", errors.New("table does not exist")
	}

	assignmentsRaw := query[setIdx+5 : whereIdx]
	condRaw := query[whereIdx+7:]

	// Parse condition
	condParts := strings.SplitN(condRaw, "=", 2)
	if len(condParts) != 2 {
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
		return "", fmt.Errorf("unknown column %s", condCol)
	}

	// Parse assignments
	assignmentList := strings.Split(assignmentsRaw, ",")
	updates := make(map[int]interface{})
	for _, a := range assignmentList {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) != 2 {
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
			return "", fmt.Errorf("unknown column %s", col)
		}
		if table.Columns[idx].Type == "INT" {
			num, err := strconv.Atoi(val)
			if err != nil {
				return "", fmt.Errorf("invalid INT value for column %s", col)
			}
			updates[idx] = num
		} else {
			updates[idx] = val
		}
	}

	var cond interface{}
	if table.Columns[condIdx].Type == "INT" {
		num, err := strconv.Atoi(condVal)
		if err != nil {
			return "", fmt.Errorf("invalid INT value for column %s", condCol)
		}
		cond = num
	} else {
		cond = condVal
	}

	updated := 0
	table.mu.Lock()
	for i, row := range table.Rows {
		if row[condIdx] == cond {
			for idx, val := range updates {
				if idx < len(row) {
					row[idx] = val
				}
			}
			table.Rows[i] = row
			updated++
		}
	}
	table.mu.Unlock()

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
