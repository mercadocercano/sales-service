.PHONY: help build test clean migrate-up migrate-down run-worker run-example-publisher run-example-consumer

help:
	@echo "Available targets:"
	@echo "  build                - Build all binaries"
	@echo "  test                 - Run tests"
	@echo "  clean                - Clean build artifacts"
	@echo "  migrate-up           - Run database migrations"
	@echo "  migrate-down         - Rollback database migrations"
	@echo "  run-worker           - Run the event worker"
	@echo "  run-example-publisher - Run example publisher"
	@echo "  run-example-consumer  - Run example consumer"

build:
	@echo "Building worker..."
	go build -o bin/worker ./cmd/worker
	@echo "Building examples..."
	go build -o bin/simple_publisher ./examples/simple_publisher
	go build -o bin/simple_consumer ./examples/simple_consumer

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

migrate-up:
	@echo "Running migrations..."
	psql $(shell cat .env | grep DB_ | xargs -I {} echo -n "-v {} ") -f migrations/001_create_event_bus_tables.up.sql

migrate-down:
	@echo "Rolling back migrations..."
	psql $(shell cat .env | grep DB_ | xargs -I {} echo -n "-v {} ") -f migrations/001_create_event_bus_tables.down.sql

run-worker: build
	@echo "Starting worker..."
	./bin/worker

run-example-publisher: build
	@echo "Running example publisher..."
	./bin/simple_publisher

run-example-consumer: build
	@echo "Running example consumer..."
	./bin/simple_consumer

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

lint: fmt vet
	@echo "Linting complete"
