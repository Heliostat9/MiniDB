package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type ColumnType string

const (
	TypeInt   ColumnType = "INT"
	TypeText  ColumnType = "TEXT"
	TypeFloat ColumnType = "FLOAT"
	TypeBool  ColumnType = "BOOL"
)

type Column struct {
	Name string
	Type ColumnType
}

// Index stores mapping from column values to row indexes for quick lookup.
type Index struct {
	Column string
	idx    int
	Values map[interface{}][]int
}

type Row []interface{}

type Table struct {
	Name    string
	Columns []Column
	Rows    []Row
	mu      sync.RWMutex
	Indexes map[string]*Index
}

var Tables = make(map[string]*Table)

func Init() error {
	if err := LoadBinaryDB(); err != nil {
		return err
	}
	return replayWAL()
}

func HandleCommand(query string) (string, error) {
	query = strings.TrimSpace(query)
	queryUpper := strings.ToUpper(query)

	switch {
	case strings.HasPrefix(queryUpper, "CREATE TABLE"):
		return handleCreateTable(query)
	case strings.HasPrefix(queryUpper, "CREATE INDEX"):
		return handleCreateIndex(query)
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
	if err := appendWAL(query); err != nil {
		return "", err
	}
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
		colType := ColumnType("TEXT")
		if len(parts) > 1 {
			colType = ColumnType(strings.ToUpper(parts[1]))
		}
		switch colType {
		case TypeInt, TypeText, TypeFloat, TypeBool:
		default:
			return "", fmt.Errorf("unknown column type %s", colType)
		}
		columns = append(columns, Column{Name: name, Type: colType})
	}

	table := &Table{
		Name:    header,
		Columns: columns,
		Rows:    []Row{},
	}

	if txCtx != nil {
		Tables[header] = table
	} else {
		dbMu.Lock()
		Tables[header] = table
		dbMu.Unlock()
	}

	returnMsg := fmt.Sprintf("Table '%s' created.", header)

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}
	if err := clearWAL(); err != nil {
		return "", err
	}
	return returnMsg, nil
}

func handleCreateIndex(query string) (string, error) {
	parts := strings.Fields(query)
	if len(parts) < 4 || strings.ToUpper(parts[1]) != "INDEX" || strings.ToUpper(parts[2]) != "ON" {
		return "", errors.New("invalid CREATE INDEX syntax")
	}
	tableName := parts[3]
	colRaw := parts[len(parts)-1]
	open := strings.Index(colRaw, "(")
	close := strings.Index(colRaw, ")")
	if open == -1 || close == -1 || close <= open {
		return "", errors.New("invalid CREATE INDEX syntax")
	}
	colName := colRaw[open+1 : close]

	var table *Table
	var exists bool
	if txCtx != nil {
		table, exists = Tables[tableName]
	} else {
		dbMu.RLock()
		table, exists = Tables[tableName]
		dbMu.RUnlock()
	}
	if !exists {
		return "", errors.New("table does not exist")
	}

	table.mu.Lock()
	err := table.createIndex(colName)
	table.mu.Unlock()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Index on %s created.", colName), nil
}

func handleInsert(query string) (string, error) {
	if err := appendWAL(query); err != nil {
		return "", err
	}
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

	var table *Table
	var exists bool
	if txCtx != nil {
		table, exists = Tables[tableName]
	} else {
		dbMu.RLock()
		table, exists = Tables[tableName]
		dbMu.RUnlock()
	}
	if !exists {
		return "", errors.New("table does not exist")
	}

	var row Row
	for i, v := range vals {
		val := strings.Trim(strings.TrimSpace(v), "'")
		if i >= len(table.Columns) {
			return "", errors.New("columns count does not match")
		}
		parsed, err := parseValue(val, table.Columns[i].Type)
		if err != nil {
			return "", fmt.Errorf("invalid %s value for column %s", table.Columns[i].Type, table.Columns[i].Name)
		}
		row = append(row, parsed)
	}

	if len(row) != len(table.Columns) {
		return "", errors.New("columns count does not match")
	}

	table.mu.Lock()
	table.Rows = append(table.Rows, row)
	idx := len(table.Rows) - 1
	table.addToIndexes(row, idx)
	table.mu.Unlock()

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}
	if err := clearWAL(); err != nil {
		return "", err
	}
	return "1 row inserted.", nil
}

func handleSelect(query string) (string, error) {
	if res, ok := resultCache.Get(query); ok {
		return res, nil
	}
	upper := strings.ToUpper(query)
	whereIdx := strings.Index(upper, " WHERE ")
	var whereCol string
	var whereVal interface{}
	if whereIdx != -1 {
		condRaw := strings.TrimSpace(query[whereIdx+7:])
		query = strings.TrimSpace(query[:whereIdx])
		parts := strings.SplitN(condRaw, "=", 2)
		if len(parts) != 2 {
			return "", errors.New("invalid WHERE syntax")
		}
		whereCol = strings.TrimSpace(parts[0])
		whereVal = strings.Trim(strings.TrimSpace(parts[1]), "'")
	}

	fromIdx := strings.Index(upper, " FROM ")
	if fromIdx == -1 {
		return "", errors.New("invalid SELECT syntax")
	}
	colsRaw := strings.TrimSpace(query[6:fromIdx])
	cols := strings.Split(colsRaw, ",")
	for i, c := range cols {
		cols[i] = strings.TrimSpace(c)
	}
	tableName := strings.Fields(query[fromIdx+6:])[0]

	var table *Table
	var exists bool
	if txCtx != nil {
		table, exists = Tables[tableName]
		if !exists {
			return "", errors.New("table does not exist")
		}
	} else {
		dbMu.RLock()
		table, exists = Tables[tableName]
		if !exists {
			dbMu.RUnlock()
			return "", errors.New("table does not exist")
		}
	}

	colIdx := make([]int, 0, len(cols))
	if len(cols) == 1 && cols[0] == "*" {
		for i := range table.Columns {
			colIdx = append(colIdx, i)
		}
		cols = make([]string, len(table.Columns))
		for i, c := range table.Columns {
			cols[i] = c.Name
		}
	} else {
		for _, c := range cols {
			idx := -1
			for i, col := range table.Columns {
				if col.Name == c {
					idx = i
					break
				}
			}
			if idx == -1 {
				if txCtx == nil {
					dbMu.RUnlock()
				}
				return "", fmt.Errorf("unknown column %s", c)
			}
			colIdx = append(colIdx, idx)
		}
	}

	var whereIdxCol int
	var whereParsed interface{}
	if whereCol != "" {
		idx := -1
		for i, c := range table.Columns {
			if c.Name == whereCol {
				idx = i
				break
			}
		}
		if idx == -1 {
			if txCtx == nil {
				dbMu.RUnlock()
			}
			return "", fmt.Errorf("unknown column %s", whereCol)
		}
		parsed, err := parseValue(fmt.Sprint(whereVal), table.Columns[idx].Type)
		if err != nil {
			if txCtx == nil {
				dbMu.RUnlock()
			}
			return "", fmt.Errorf("invalid %s value for column %s", table.Columns[idx].Type, whereCol)
		}
		whereIdxCol = idx
		whereParsed = parsed
	}

	var builder strings.Builder
	builder.WriteString(strings.Join(cols, "\t") + "\n")

	table.mu.RLock()
	var rowIndexes []int
	if whereCol != "" {
		if idx, ok := table.Indexes[whereCol]; ok {
			if rows, ok2 := idx.Values[whereParsed]; ok2 {
				rowIndexes = append(rowIndexes, rows...)
			}
		}
	}
	if rowIndexes == nil {
		rowIndexes = make([]int, len(table.Rows))
		for i := range table.Rows {
			rowIndexes[i] = i
		}
	}
	for _, rid := range rowIndexes {
		if rid >= len(table.Rows) {
			continue
		}
		row := table.Rows[rid]
		if whereCol != "" && row[whereIdxCol] != whereParsed {
			continue
		}
		strVals := make([]string, len(colIdx))
		for i, idx := range colIdx {
			strVals[i] = fmt.Sprint(row[idx])
		}
		builder.WriteString(strings.Join(strVals, "\t") + "\n")
	}
	table.mu.RUnlock()
	if txCtx == nil {
		dbMu.RUnlock()
	}

	res := builder.String()
	resultCache.Add(query, res)
	return res, nil
}

func handleUpdate(query string) (string, error) {
	if err := appendWAL(query); err != nil {
		return "", err
	}
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

	var table *Table
	var exists bool
	if txCtx != nil {
		table, exists = Tables[tableName]
	} else {
		dbMu.RLock()
		table, exists = Tables[tableName]
		dbMu.RUnlock()
	}
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
		parsed, err := parseValue(val, table.Columns[idx].Type)
		if err != nil {
			return "", fmt.Errorf("invalid %s value for column %s", table.Columns[idx].Type, col)
		}
		updates[idx] = parsed
	}

	var cond interface{}
	parsedCond, err := parseValue(condVal, table.Columns[condIdx].Type)
	if err != nil {
		return "", fmt.Errorf("invalid %s value for column %s", table.Columns[condIdx].Type, condCol)
	}
	cond = parsedCond

	updated := 0
	table.mu.Lock()
	for i, row := range table.Rows {
		if row[condIdx] == cond {
			for idx, val := range updates {
				if idx < len(row) {
					row[idx] = val
				}
			}
			old := table.Rows[i]
			table.Rows[i] = row
			table.updateIndexes(old, row, i)
			updated++
		}
	}
	table.mu.Unlock()

	if err := SaveBinaryDB(); err != nil {
		return "", err
	}
	if err := clearWAL(); err != nil {
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

func parseValue(val string, ct ColumnType) (interface{}, error) {
	switch ct {
	case TypeInt:
		return strconv.Atoi(val)
	case TypeFloat:
		return strconv.ParseFloat(val, 64)
	case TypeBool:
		lower := strings.ToLower(val)
		if lower == "true" || lower == "1" {
			return true, nil
		}
		if lower == "false" || lower == "0" {
			return false, nil
		}
		return nil, fmt.Errorf("invalid BOOL value")
	default:
		return val, nil
	}
}

func (t *Table) createIndex(column string) error {
	idx := -1
	for i, c := range t.Columns {
		if c.Name == column {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("unknown column %s", column)
	}
	m := make(map[interface{}][]int)
	for i, row := range t.Rows {
		v := row[idx]
		m[v] = append(m[v], i)
	}
	if t.Indexes == nil {
		t.Indexes = make(map[string]*Index)
	}
	t.Indexes[column] = &Index{Column: column, idx: idx, Values: m}
	return nil
}

func (t *Table) addToIndexes(row Row, rowIdx int) {
	for _, idx := range t.Indexes {
		val := row[idx.idx]
		idx.Values[val] = append(idx.Values[val], rowIdx)
	}
}

func (t *Table) updateIndexes(oldRow, newRow Row, rowIdx int) {
	for _, idx := range t.Indexes {
		ov := oldRow[idx.idx]
		nv := newRow[idx.idx]
		if ov == nv {
			continue
		}
		arr := idx.Values[ov]
		for i := range arr {
			if arr[i] == rowIdx {
				idx.Values[ov] = append(arr[:i], arr[i+1:]...)
				break
			}
		}
		idx.Values[nv] = append(idx.Values[nv], rowIdx)
	}
}
