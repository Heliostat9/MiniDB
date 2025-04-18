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
	dbVersion   = uint8(1)
)

const binaryDBFile = "data.mdb"

func SaveBinaryDB() error {
	file, err := os.Create(binaryDBFile)

	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(magicHeader); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, dbVersion); err != nil {
		return err
	}

	for _, table := range Tables {
		err := writeTable(file, table)
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
	defer file.Close()

	header := make([]byte, len(magicHeader))
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	if !bytes.Equal(header, magicHeader) {
		return fmt.Errorf("invalid file format: bad magic header")
	}

	var version uint8
	if err := binary.Read(file, binary.LittleEndian, &version); err != nil {
		return fmt.Errorf("reding version: %w", err)
	}

	if version > dbVersion {
		return fmt.Errorf("unsupported db version: %d", version)
	}

	Tables = make(map[string]*Table)
	for {
		table, err := readTable(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		Tables[table.Name] = table
	}

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
		colLen := uint8(len(col))
		if err := binary.Write(w, binary.LittleEndian, colLen); err != nil {
			return err
		}
		if _, err := w.Write([]byte(col)); err != nil {
			return err
		}
	}

	// WRITE: Rows

	rowCount := uint32(len(table.Rows))
	if err := binary.Write(w, binary.LittleEndian, rowCount); err != nil {
		return err
	}

	for _, row := range table.Rows {
		for _, data := range row {
			dataLen := uint16(len(data))
			if err := binary.Write(w, binary.LittleEndian, dataLen); err != nil {
				return err
			}

			if _, err := w.Write([]byte(data)); err != nil {
				return err
			}
		}
	}

	return nil
}

func readTable(r io.Reader) (*Table, error) {
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

	columns := make([]string, 0, colCount)

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

		columns = append(columns, columnName)
	}

	//READ: Rows
	var rowCount uint32
	err = binary.Read(r, binary.LittleEndian, &rowCount)

	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, rowCount)

	for i := 0; i < int(rowCount); i++ {
		row := make([]string, 0, colCount)

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
