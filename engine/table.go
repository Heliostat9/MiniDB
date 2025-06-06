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

var Tables = make(map[string]*Table)

const dbFile = "data.db"

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
	default:
		return "", errors.New("unsupported command")
	}
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

	Tables[header] = table
	returnMsg := fmt.Sprintf("Table '%s' created.", header)

	return returnMsg, SaveBinaryDB()
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
	table, exists := Tables[tableName]
	if !exists {
		return "", errors.New("table does not exist")
	}

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

	if len(row) != len(table.Columns) {
		return "", errors.New("columns count does not match")
	}

	table.Rows = append(table.Rows, row)
	return "1 row inserted.", SaveBinaryDB()
}

func handleSelect(query string) (string, error) {
	tokens := strings.Fields(query)
	if len(tokens) < 4 || tokens[1] != "*" || tokens[2] != "FROM" {
		return "", errors.New("only SELECT * FROM <table> supported for now")
	}

	tableName := tokens[3]
	table, exists := Tables[tableName]

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
	table, exists := Tables[tableName]
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
		if c == condCol {
			condIdx = i
			break
		}
	}
	if condIdx == -1 {
		return "", fmt.Errorf("unknown column %s", condCol)
	}

	// Parse assignments
	assignmentList := strings.Split(assignmentsRaw, ",")
	updates := make(map[int]string)
	for _, a := range assignmentList {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) != 2 {
			return "", errors.New("invalid SET syntax")
		}
		col := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "'")

		idx := -1
		for i, c := range table.Columns {
			if c == col {
				idx = i
				break
			}
		}
		if idx == -1 {
			return "", fmt.Errorf("unknown column %s", col)
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

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d rows updated.", updated), nil
}
