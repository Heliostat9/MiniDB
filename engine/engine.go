package engine

import (
	"errors"
	"strings"
)

func Execute(query string) (string, error) {
	query = strings.TrimSpace(strings.ToUpper(query))

	if strings.HasPrefix(query, "CREATE TABLE") {
		return handleCreateTable(query)
	}

	if strings.HasPrefix(query, "INSERT INTO") {
		return handleInsert(query)
	}

	if strings.HasPrefix(query, "SELECT") {
		return handleSelect(query)
	}

	return "", errors.New("unsupported query")
}
