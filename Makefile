.PHONY: help build run test clean proto docker-build docker-run docker-stop

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "🚀 Development (Recommended for local work):"
	@echo "  dev-full     - Start DB + run with hot reload (one command)"
	@echo "  dev-db       - Start only DynamoDB Local for development"
	@echo "  dev-run      - Run server locally with go run (fast iteration)"
	@echo "  dev          - Run with hot reload (requires air: go install github.com/air-verse/air@latest)"
	@echo "  dev-db-stop  - Stop development DynamoDB"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  test         - Run tests (includes setup and cleanup)"
	@echo "  test-env     - Setup test environment only"
	@echo "  test-setup   - Setup test environment (DynamoDB + table)"
	@echo "  test-cleanup - Clean up test environment"
	@echo ""
	@echo "🏗️  Building & Running:"
	@echo "  build        - Build the gRPC server"
	@echo "  run          - Run the server locally (requires dev-db)"
	@echo "  clean        - Clean build artifacts"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build    - Build Docker image (includes code building)"
	@echo "  docker-build-ci - Build Docker image for CI (uses pre-built binary)"
	@echo "  docker-run      - Start DynamoDB only (for local development)"
	@echo "  docker-stop     - Stop DynamoDB"
	@echo ""
	@echo "📋 Utilities:"
	@echo "  proto        - Generate protobuf code"
	@echo "  env-status   - Check all environment statuses"
	@echo "  cleanup-orphans - Clean up orphaned containers and networks"

# Build the server
build:
	@echo "Building gRPC server..."
	@mkdir -p bin
	go build -o bin/store-service ./cmd/server

# Build the server with version information for CI
build-ci:
	@echo "Building gRPC server for CI with version info..."
	@mkdir -p bin
	@if [ -n "$(VERSION)" ] && [ -n "$(COMMIT)" ]; then \
		echo "Building with version: $(VERSION)-$(COMMIT)"; \
		go build -ldflags="-X main.version=$(VERSION) -X main.buildCommit=$(COMMIT) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/store-service ./cmd/server; \
	else \
		echo "Building without version info (use VERSION and COMMIT env vars)"; \
		go build -o bin/store-service ./cmd/server; \
	fi

# Run the server locally (requires DynamoDB to be running)
run: build
	@echo "Running gRPC server locally..."
	@echo "Make sure DynamoDB is running with: make dev-db"
	@DYNAMODB_TABLE_NAME=store-items AWS_REGION=us-east-1 DYNAMODB_ENDPOINT=http://localhost:8000 ./bin/store-service

# Run the server locally without building (faster iteration)
dev-run:
	@echo "Running gRPC server with go run..."
	@echo "Make sure DynamoDB is running with: make dev-db"
	@DYNAMODB_TABLE_NAME=store-items AWS_REGION=us-east-1 DYNAMODB_ENDPOINT=http://localhost:8000 go run ./cmd/server

# Run tests
test: test-setup
	@echo "Running tests..."
	go test -v ./internal/...
	@echo "Cleaning up test environment..."
	@make test-cleanup

# Setup test environment
test-setup:
	@echo "Setting up test environment..."
	@docker compose -f docker-compose.test.yml up -d --remove-orphans dynamodb-local-test
	@echo "Waiting for DynamoDB Local to be ready..."
	@sleep 5
	@echo "Creating test table..."
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb create-table \
		--endpoint-url=http://localhost:8001 \
		--region us-east-1 \
		--table-name store-items-test \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--provisioned-throughput \
			ReadCapacityUnits=5,WriteCapacityUnits=5 >/dev/null 2>&1 || echo "Table already exists or creation failed"
	@echo "Waiting for table to be active..."
	@sleep 3

# Cleanup test environment
test-cleanup:
	@echo "Cleaning up test environment..."
	@docker compose -f docker-compose.test.yml down --remove-orphans

# Setup test environment only (useful for development)
test-env: test-setup
	@echo "Test environment is ready. Run 'make test' to run tests or 'make test-cleanup' to clean up."

# Check if development environment is running
dev-status:
	@echo "Checking development environment status..."
	@docker compose ps

# Check if test environment is running
test-status:
	@echo "Checking test environment status..."
	@docker compose -f docker-compose.test.yml ps

# Show comprehensive environment status
env-status:
	@echo "Checking all environment statuses..."
	@echo "Development DynamoDB:"
	@docker compose ps 2>/dev/null || echo "Not running"
	@echo "Test DynamoDB:"
	@docker compose -f docker-compose.test.yml ps 2>/dev/null || echo "Not running"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/store.proto

# Build Docker image (includes code building)
docker-build:
	@echo "Building Docker image with code compilation..."
	docker build -t store-service:latest .

# Build Docker image for CI (uses pre-built binary)
docker-build-ci:
	@echo "Building Docker image for CI (using pre-built binary)..."
	@echo "Make sure to run 'make build' first to create bin/store-service"
	docker build -f Dockerfile.ci -t store-service:latest .

# Start only DynamoDB for local development (app runs with go run)
dev-db:
	@echo "Starting DynamoDB Local for development..."
	@docker compose up -d --remove-orphans dynamodb-local
	@echo "Waiting for DynamoDB to be ready..."
	@sleep 3
	@echo "Creating development table..."
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb create-table \
		--endpoint-url=http://localhost:8000 \
		--region us-east-1 \
		--table-name store-items \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--provisioned-throughput \
			ReadCapacityUnits=5,WriteCapacityUnits=5 >/dev/null 2>&1 || echo "Table already exists or creation failed"
	@echo "DynamoDB is ready! Run 'make dev-run' to start the server locally."

# Stop development DynamoDB
dev-db-stop:
	@echo "Stopping DynamoDB Local..."
	@docker compose down --remove-orphans

# Clean up all orphaned containers and networks
cleanup-orphans:
	@echo "Cleaning up orphaned containers and networks..."
	@docker compose down --remove-orphans --volumes 2>/dev/null || true
	@docker compose -f docker-compose.test.yml down --remove-orphans --volumes 2>/dev/null || true
	@echo "Orphaned containers and networks cleaned up."
