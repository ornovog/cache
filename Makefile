# Go Caching Layer Makefile

.PHONY: all build test bench clean run format lint help

# Default target
all: format lint test

# Build the application
build:
	@echo "Building the application..."
	@go build -o bin/caching-layer .

# Run the demo application
run:
	@echo "Running the caching layer demo..."
	@go run .

# Run all tests
test:
	@echo "Running tests..."
	@go test -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run specific test
test-specific:
	@echo "Running specific test (use TEST=TestName)..."
	@go test -run $(TEST) -v

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem

# Run performance comparison benchmarks
bench-compare:
	@echo "Running performance comparison benchmarks..."
	@go test -bench=BenchmarkDirectFunction -benchmem
	@go test -bench=BenchmarkCachedFunctionCold -benchmem
	@go test -bench=BenchmarkCachedFunctionWarm -benchmem
	@go test -bench=BenchmarkHighConcurrency -benchmem

# Format code
format:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run race detection tests
test-race:
	@echo "Running tests with race detection..."
	@go test -race -v

# Generate test data for performance analysis
test-memory:
	@echo "Running memory usage tests..."
	@go test -memprofile=mem.prof -bench=.
	@go tool pprof mem.prof

# Generate CPU profile for performance analysis
test-cpu:
	@echo "Running CPU profiling tests..."
	@go test -cpuprofile=cpu.prof -bench=.
	@go tool pprof cpu.prof

# Run comprehensive test suite
test-all: test test-race test-coverage

# Display help
help:
	@echo "Available targets:"
	@echo "  all           - Format, lint, and test (default)"
	@echo "  build         - Build the application"
	@echo "  run           - Run the demo application"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-specific - Run specific test (use TEST=TestName)"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-memory   - Run memory profiling"
	@echo "  test-cpu      - Run CPU profiling"
	@echo "  test-all      - Run comprehensive test suite"
	@echo "  bench         - Run all benchmarks"
	@echo "  bench-compare - Run performance comparison benchmarks"
	@echo "  format        - Format code with go fmt"
	@echo "  lint          - Lint code with go vet"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install/update dependencies"
	@echo "  help          - Display this help message" 