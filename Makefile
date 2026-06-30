.PHONY: build run dev test clean docker demo all

# Build the application
build:
	go build -o bin/cryptojackal ./cmd/cryptojackal
	go build -o bin/demo ./cmd/demo

# Run the application
run:
	go run ./cmd/cryptojackal

# Run both backend and frontend in dev mode (parallel)
dev:
	@echo "Starting backend on :8080 and frontend on :3000..."
	@trap 'kill %1 %2 2>/dev/null; exit' INT; \
	go run ./cmd/cryptojackal & \
	cd web && npm run dev & \
	wait

# Run the demo
demo:
	go run ./cmd/demo

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Build Docker image
docker:
	docker build -t cryptojackal .

# Run with Docker Compose
docker-up:
	docker compose up -d

# Stop Docker Compose
docker-down:
	docker compose down

# View logs
logs:
	docker compose logs -f cryptojackal

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Generate mocks (if needed)
generate:
	go generate ./...
