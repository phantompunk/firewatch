.PHONY: help clean build build-prod dev run

# Set the default goal
.DEFAULT_GOAL := help

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## dev: run the server with live reload (requires air - go install github.com/cosmtrek/air@latest)
dev:
	air

## build: Build the server binary
build:
	go build -o ./tmp/firewatch ./cmd/server

## run: run the server (development mode - no SMTP)
run: build
	./tmp/firewatch

## clean: clean build artifacts
clean:
	rm -f ./tmp/firewatch

## build-prod: build for production (static binary)
build-prod:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o firewatch ./cmd/server
