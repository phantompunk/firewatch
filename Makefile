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

## secrets: Generate local secret key files for Docker
secrets:
		@mkdir -p secrets
		@[ -f secrets/session_secret ]					|| openssl rand -out secrets/session_secret 32
		@[ -f secrets/settings_encryption_key ] || openssl rand -out secrets/settings_encryption_key 32
		@[ -f secrets/email_hmac_key ]					|| openssl rand -out secrets/email_hmac_key 32
		@echo "Secret files ready in ./secrets/"
