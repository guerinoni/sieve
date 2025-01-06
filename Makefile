# Variables
PKG := ./...            # Packages to test
COVERAGE_FILE := coverage.out

# Default target
.DEFAULT_GOAL := help

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -race $(PKG) -v

# Run tests with escape analysis
.PHONY: test-escape
test-escape:
	@echo "Running tests with escape analysis..."
	go test -gcflags="-m" $(PKG)

# Run tests and generate coverage report
.PHONY: coverage
coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=$(COVERAGE_FILE) $(PKG)
	@echo "Coverage details:"
	go tool cover -func=$(COVERAGE_FILE)

# View HTML coverage report
.PHONY: coverage-html
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE)

# Benchmark tests
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. $(PKG)

# Help menu
.PHONY: help
help:
	@echo "Makefile Targets:"
	@echo "  test             - Run tests"
	@echo "  bench            - Run benchmarks"
	@echo "  test-escape      - Run tests with escape analysis"
	@echo "  coverage         - Run tests with coverage reporting"
	@echo "  coverage-html    - View HTML coverage report"
