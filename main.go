package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"minisql/engine"
)

func main() {
	err := engine.Init()
	if err != nil {
		fmt.Println("Error loading DB:", err)
		return
	}

	fmt.Println("Welcome to MiniSQL")
	fmt.Println("Type SQL statements (end with semicolon ';'). Type 'exit;' to quit")

	scanner := bufio.NewScanner(os.Stdin)
	queryBuffer := ""

	for {
		fmt.Print(">> ")
		scanner.Scan()
		line := scanner.Text()
		queryBuffer += " " + line

		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			query := strings.TrimSuffix(queryBuffer, ";")
			queryBuffer = ""

			if strings.TrimSpace(query) == "exit" {
				break
			}

			result, err := engine.Execute(query)

			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println(result)
			}

		}
	}
}
