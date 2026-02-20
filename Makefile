.PHONY: run build migrate test lint generate

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

migrate:
	go run ./cmd/migrate

test:
	go test ./...

lint:
	golangci-lint run ./...

generate:
	sqlc generate
