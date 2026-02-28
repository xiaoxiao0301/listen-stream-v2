.PHONY: help build test run clean

help:
	@echo "Available commands:"
	@echo "  build       - Build the service"
	@echo "  test        - Run tests"
	@echo "  run         - Run the service"
	@echo "  clean       - Clean build artifacts"

build:
	@echo "Building proxy-svc..."
	go build -o bin/proxy-svc ./cmd/main.go

test:
	@echo "Running tests..."
	go test -v -race ./...

run:
	@echo "Starting proxy-svc..."
	go run cmd/main.go

clean:
	@echo "Cleaning..."
	rm -rf bin/
