.PHONY: build run test clean deps install

# Build the application
build:
	go build -o truck-inventory main.go

# Run the application
run: build
	./truck-inventory

# Run tests
test:
	go test ./...

# Test SLM-based listing extraction (requires SLM running or SLM_ENABLED=false for fallback)
test-extraction:
	go run ./cmd/test-extraction

# Download dependencies
deps:
	go mod download
	go mod tidy

# Install dependencies and build
install: deps build

# Clean build artifacts
clean:
	rm -f truck-inventory truck-inventory.exe
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Create .env from example
setup-env:
	cp .env.example .env
	@echo "✅ Created .env file. Please edit it with your credentials."
