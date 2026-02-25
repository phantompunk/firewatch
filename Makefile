.PHONY: run build migrate test lint generate

# Set the default goal
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Usage: \n make [command]"
	@echo ""
	@echo "Commands:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: Run the application
run:
	@echo "Running the application..."
	@go run ./cmd/server

## dev: Run locally with Air
dev:
	@echo "Starting local dev with Air..."
	@set -a && source .env.local && set +a && air

## up: Docker compose up
up:
	@echo "Starting Docker environment..."
	@docker-compose --env-file .env.docker up -d --build

## down: Docker compose down
down:
	@echo "Stopping Docker environment..."
	@docker-compose down

## build: Build the application
build:
	@echo "Building the application..."
	@go build -o bin/server ./cmd/server

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/* tmp/*
