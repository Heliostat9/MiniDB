.PHONY: build test lint coverage fmt

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

coverage:
	go test -coverprofile=coverage.out ./...

fmt:
	gofmt -w $(shell go list -f '{{.Dir}}' ./...)
