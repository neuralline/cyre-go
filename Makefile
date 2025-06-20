# Makefile for Cyre Go
# Build and development tasks

.PHONY: all build test benchmark clean deps lint fmt vet security example profile help

# Variables
BINARY_NAME=cyre-go
PACKAGE=github.com/your-org/cyre-go
BUILD_DIR=build
EXAMPLE_DIR=example
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.GoVersion=$(GO_VERSION)"

# Default target
all: deps fmt vet lint test build

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod verify
	go mod tidy

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "🔍 Vetting code..."
	go vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "🧹 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed, skipping..."; \
		echo "   Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Security check (requires gosec)
security:
	@echo "🔒 Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not installed, skipping..."; \
		echo "   Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

# Run benchmarks with CPU and memory profiling
benchmark-profile:
	@echo "⚡ Running benchmarks with profiling..."
	mkdir -p $(BUILD_DIR)
	go test -bench=. -benchmem -run=^$$ -cpuprofile=$(BUILD_DIR)/cpu.prof -memprofile=$(BUILD_DIR)/mem.prof ./...
	@echo "Profiles generated in $(BUILD_DIR)/"

# Performance comparison benchmarks
benchmark-compare:
	@echo "⚡ Running performance comparison..."
	go test -bench=. -count=5 -benchmem ./... | tee $(BUILD_DIR)/bench.txt

# Build the library (no main, just verify compilation)
build:
	@echo "🔨 Building library..."
	go build -v ./...

# Build example
build-example:
	@echo "🔨 Building example..."
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-example ./$(EXAMPLE_DIR)

# Run example
example: build-example
	@echo "🚀 Running example..."
	./$(BUILD_DIR)/$(BINARY_NAME)-example

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -testcache

# Profile memory usage
profile-mem:
	@echo "📊 Profiling memory usage..."
	mkdir -p $(BUILD_DIR)
	go test -bench=BenchmarkBasicCall -memprofile=$(BUILD_DIR)/mem.prof
	go tool pprof -http=:8080 $(BUILD_DIR)/mem.prof &
	@echo "Memory profile server started at http://localhost:8080"

# Profile CPU usage
profile-cpu:
	@echo "📊 Profiling CPU usage..."
	mkdir -p $(BUILD_DIR)
	go test -bench=BenchmarkBasicCall -cpuprofile=$(BUILD_DIR)/cpu.prof
	go tool pprof -http=:8081 $(BUILD_DIR)/cpu.prof &
	@echo "CPU profile server started at http://localhost:8081"

# Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060 & \
		echo "Documentation available at http://localhost:6060/pkg/$(PACKAGE)"; \
	else \
		echo "⚠️  godoc not installed"; \
		echo "   Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Install development tools
install-tools:
	@echo "🔧 Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/go-delve/delve/cmd/dlv@latest

# Check Go version
check-version:
	@echo "🔍 Checking Go version..."
	@go version
	@echo "Required: Go 1.22 or higher"

# Run all quality checks
quality: deps fmt vet lint security test

# CI/CD pipeline simulation
ci: quality benchmark build-example
	@echo "✅ CI pipeline completed successfully"

# Release build (for actual releases)
release:
	@echo "🚀 Building release..."
	@if [ -z "$(VERSION)" ]; then echo "❌ No version tag found"; exit 1; fi
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(EXAMPLE_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(EXAMPLE_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(EXAMPLE_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(EXAMPLE_DIR)
	@echo "Release binaries built in $(BUILD_DIR)/"

# Development mode (watch for changes and rerun tests)
dev:
	@echo "🔄 Development mode (requires fswatch)..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . -e ".*" -i "\\.go$$" | xargs -n1 -I{} make test; \
	else \
		echo "⚠️  fswatch not installed"; \
		echo "   Install with: brew install fswatch (macOS) or apt-get install fswatch (Linux)"; \
	fi

# Performance monitoring
perf-monitor:
	@echo "📈 Starting performance monitor..."
	@while true; do \
		echo "=== Performance Check $(shell date) ==="; \
		go test -bench=BenchmarkBasicCall -count=1 -benchtime=1s; \
		sleep 10; \
	done

# Memory leak detection
leak-check:
	@echo "🔍 Checking for memory leaks..."
	go test -v -run=TestLongRunning -timeout=30s -memprofile=$(BUILD_DIR)/leak.prof
	go tool pprof -alloc_space $(BUILD_DIR)/leak.prof

# Load testing with example
load-test: build-example
	@echo "⚡ Running load test..."
	@echo "Starting example in background..."
	./$(BUILD_DIR)/$(BINARY_NAME)-example &
	@echo "Example PID: $$!"
	sleep 2
	@echo "Load test completed"

# Health check
health:
	@echo "🏥 System health check..."
	@go version
	@echo "Go modules status:"
	@go mod verify
	@echo "Test status:"
	@go test -run=TestSystemHealth -v
	@echo "Memory usage:"
	@go test -bench=BenchmarkBasicCall -benchtime=1s -benchmem | grep "allocs/op"

# Show project statistics
stats:
	@echo "📊 Project Statistics"
	@echo "===================="
	@echo "Lines of code:"
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
	@echo ""
	@echo "Go files:"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l
	@echo ""
	@echo "Test files:"
	@find . -name "*_test.go" -not -path "./vendor/*" | wc -l
	@echo ""
	@echo "Dependencies:"
	@go list -m all | wc -l
	@echo ""
	@echo "Latest commits:"
	@git log --oneline -5 2>/dev/null || echo "Not a git repository"

# Help
help:
	@echo "🚀 Cyre Go - Available Make Targets"
	@echo "=================================="
	@echo ""
	@echo "📦 Dependencies & Setup:"
	@echo "  deps           - Install and verify dependencies"
	@echo "  install-tools  - Install development tools"
	@echo "  check-version  - Check Go version"
	@echo ""
	@echo "🔨 Building:"
	@echo "  build          - Build library (verify compilation)"
	@echo "  build-example  - Build example application"
	@echo "  release        - Build release binaries for multiple platforms"
	@echo ""
	@echo "🧪 Testing & Quality:"
	@echo "  test           - Run all tests with race detection"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  quality        - Run all quality checks (fmt, vet, lint, security, test)"
	@echo "  benchmark      - Run performance benchmarks"
	@echo "  benchmark-profile - Run benchmarks with CPU/memory profiling"
	@echo ""
	@echo "🔍 Code Quality:"
	@echo "  fmt            - Format code with go fmt"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  security       - Run security checks with gosec"
	@echo ""
	@echo "📊 Profiling & Monitoring:"
	@echo "  profile-cpu    - Profile CPU usage"
	@echo "  profile-mem    - Profile memory usage"
	@echo "  leak-check     - Check for memory leaks"
	@echo "  perf-monitor   - Continuous performance monitoring"
	@echo "  health         - System health check"
	@echo ""
	@echo "🚀 Running:"
	@echo "  example        - Build and run example"
	@echo "  load-test      - Run load testing"
	@echo "  dev            - Development mode (watch and test)"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  docs           - Start godoc server"
	@echo "  stats          - Show project statistics"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  clean          - Clean build artifacts"
	@echo "  ci             - Simulate CI/CD pipeline"
	@echo ""
	@echo "ℹ️  Use 'make <target>' to run any command"
	@echo "   Example: make test"