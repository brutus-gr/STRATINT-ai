.PHONY: run fmt lint tidy test ci

run:
	go run ./cmd/server

fmt:
	gofmt -w ./

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy

ci: fmt lint test
