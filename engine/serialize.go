package engine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var (
	magicHeader = []byte("MYDB")
	dbVersion   = uint8(2)
)

const binaryDBFile = "data.mdb"

// Limit maximum rows per table when loading to avoid excessive memory usage
const maxRowCount = 1_000_000

func SaveBinaryDB() error {
	dbMu.RLock()
	defer dbMu.RUnlock()

	file, err := os.OpenFile(binaryDBFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)

	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(magicHeader); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, dbVersion); err != nil {
		return err
	}

	for _, table := range Tables {
		table.mu.RLock()
		err := writeTable(file, table)
		table.mu.RUnlock()
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadBinaryDB() error {
	file, err := os.Open(binaryDBFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	header := make([]byte, len(magicHeader))
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	if !bytes.Equal(header, magicHeader) {
		return fmt.Errorf("invalid file format: bad magic header")
	}

	var version uint8
	if err := binary.Read(file, binary.LittleEndian, &version); err != nil {
		return fmt.Errorf("reading version: %w", err)
	}

	if version > dbVersion {
		return fmt.Errorf("unsupported db version: %d", version)
	}

	newTables := make(map[string]*Table)
	for {
		var table *Table
		var err error
		if version == 1 {
			table, err = readTableV1(file)
		} else {
			table, err = readTableV2(file)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		newTables[table.Name] = table
	}

	dbMu.Lock()
	Tables = newTables
	dbMu.Unlock()

	return nil
}

func writeTable(w io.Writer, table *Table) error {
	// WRITE: Name table
	nameLen := uint8(len(table.Name))
	if err := binary.Write(w, binary.LittleEndian, nameLen); err != nil {
		return err
	}
	if _, err := w.Write([]byte(table.Name)); err != nil {
		return err
	}

	// WRITE: Cols
	colCount := uint8(len(table.Columns))
	if err := binary.Write(w, binary.LittleEndian, colCount); err != nil {
		return err
	}

	for _, col := range table.Columns {
		nameLen := uint8(len(col.Name))
		if err := binary.Write(w, binary.LittleEndian, nameLen); err != nil {
			return err
		}
		if _, err := w.Write([]byte(col.Name)); err != nil {
			return err
		}

		typeLen := uint8(len(col.Type))
		if err := binary.Write(w, binary.LittleEndian, typeLen); err != nil {
			return err
		}
		if _, err := w.Write([]byte(col.Type)); err != nil {
			return err
		}
	}

	// WRITE: Rows

	rowCount := uint32(len(table.Rows))
	if err := binary.Write(w, binary.LittleEndian, rowCount); err != nil {
		return err
	}

	for _, row := range table.Rows {
		for _, val := range row {
			str := fmt.Sprint(val)
			dataLen := uint16(len(str))
			if err := binary.Write(w, binary.LittleEndian, dataLen); err != nil {
				return err
			}

			if _, err := w.Write([]byte(str)); err != nil {
				return err
			}
		}
	}

	return nil
}

func readTableV1(r io.Reader) (*Table, error) {
	// READ: Table name
	var nameLen uint8
	err := binary.Read(r, binary.LittleEndian, &nameLen)
	if err != nil {
		return nil, err
	}

	nameBytes := make([]byte, nameLen)
	_, err = io.ReadFull(r, nameBytes)
	if err != nil {
		return nil, err
	}

	tableName := string(nameBytes)

	// READ: Columns

	var colCount uint8
	err = binary.Read(r, binary.LittleEndian, &colCount)

	if err != nil {
		return nil, err
	}

	columns := make([]Column, 0, colCount)

	for i := 0; i < int(colCount); i++ {
		var colLen uint8
		err = binary.Read(r, binary.LittleEndian, &colLen)
		if err != nil {
			return nil, err
		}

		colBytes := make([]byte, colLen)
		_, err = io.ReadFull(r, colBytes)
		if err != nil {
			return nil, err
		}

		columnName := string(colBytes)

		columns = append(columns, Column{Name: columnName, Type: TypeText})
	}

	// READ: Rows
	var rowCount uint32
	err = binary.Read(r, binary.LittleEndian, &rowCount)

	if err != nil {
		return nil, err
	}

	if rowCount > maxRowCount {
		return nil, fmt.Errorf("row count %d exceeds limit", rowCount)
	}

	rows := make([]Row, 0, rowCount)

	for i := 0; i < int(rowCount); i++ {
		row := make(Row, 0, colCount)

		for j := 0; j < int(colCount); j++ {
			var valLen uint16
			err = binary.Read(r, binary.LittleEndian, &valLen)
			if err != nil {
				return nil, err
			}

			valBytes := make([]byte, valLen)
			_, err = io.ReadFull(r, valBytes)
			if err != nil {
				return nil, err
			}

			row = append(row, string(valBytes))
		}

		rows = append(rows, row)
	}

	return &Table{
		Name:    tableName,
		Columns: columns,
		Rows:    rows,
	}, nil
}

func readTableV2(r io.Reader) (*Table, error) {
	// READ: Table name
	var nameLen uint8
	err := binary.Read(r, binary.LittleEndian, &nameLen)
	if err != nil {
		return nil, err
	}

	nameBytes := make([]byte, nameLen)
	_, err = io.ReadFull(r, nameBytes)
	if err != nil {
		return nil, err
	}

	tableName := string(nameBytes)

	// READ: Columns

	var colCount uint8
	err = binary.Read(r, binary.LittleEndian, &colCount)

	if err != nil {
		return nil, err
	}

	columns := make([]Column, 0, colCount)

	for i := 0; i < int(colCount); i++ {
		var colLen uint8
		err = binary.Read(r, binary.LittleEndian, &colLen)
		if err != nil {
			return nil, err
		}

		colBytes := make([]byte, colLen)
		_, err = io.ReadFull(r, colBytes)
		if err != nil {
			return nil, err
		}

		var typeLen uint8
		err = binary.Read(r, binary.LittleEndian, &typeLen)
		if err != nil {
			return nil, err
		}

		typeBytes := make([]byte, typeLen)
		_, err = io.ReadFull(r, typeBytes)
		if err != nil {
			return nil, err
		}

		columns = append(columns, Column{Name: string(colBytes), Type: ColumnType(string(typeBytes))})
	}

	// READ: Rows
	var rowCount uint32
	err = binary.Read(r, binary.LittleEndian, &rowCount)

	if err != nil {
		return nil, err
	}

	if rowCount > maxRowCount {
		return nil, fmt.Errorf("row count %d exceeds limit", rowCount)
	}

	rows := make([]Row, 0, rowCount)

	for i := 0; i < int(rowCount); i++ {
		row := make(Row, 0, colCount)

		for j := 0; j < int(colCount); j++ {
			var valLen uint16
			err = binary.Read(r, binary.LittleEndian, &valLen)
			if err != nil {
				return nil, err
			}

			valBytes := make([]byte, valLen)
			_, err = io.ReadFull(r, valBytes)
			if err != nil {
				return nil, err
			}

			valStr := string(valBytes)
			parsed, err := parseValue(valStr, columns[j].Type)
			if err != nil {
				row = append(row, valStr)
			} else {
				row = append(row, parsed)
			}
		}

		rows = append(rows, row)
	}

	return &Table{
		Name:    tableName,
		Columns: columns,
		Rows:    rows,
	}, nil
}
