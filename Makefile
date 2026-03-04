.PHONY: help run dev up down build clean secrets

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
	@docker-compose up -d --build

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
	@rm -rf bin/* tmp/main

## secrets: Generate local secret files in ./tmp/ for local Docker
secrets:
	@mkdir -p tmp
	@[ -f tmp/session_secret ]             || openssl rand -out tmp/session_secret 32
	@[ -f tmp/settings_encryption_key ]    || openssl rand -out tmp/settings_encryption_key 32
	@[ -f tmp/email_hmac_key ]             || openssl rand -out tmp/email_hmac_key 32
	@echo "Secret files ready in ./tmp/"
