.PHONY: build run clean dev

# Build the server binary
build:
	go build -o firewatch-server ./cmd/server

# Run the server (development mode - no SMTP)
run: build
	./firewatch-server

# Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Clean build artifacts
clean:
	rm -f firewatch-server

# Build for production (static binary)
build-prod:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o firewatch-server ./cmd/server
