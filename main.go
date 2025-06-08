package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"minisql/engine"
)

func main() {
	listen := flag.String("listen", "", "start HTTP server on this address")
	flag.Parse()

	if err := engine.Init(); err != nil {
		fmt.Println("Error loading DB:", err)
		return
	}

	if *listen != "" {
		http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			res, err := engine.Execute(string(data))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(res))
		})
		log.Printf("Listening on %s", *listen)
		log.Fatal(http.ListenAndServe(*listen, nil))
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
