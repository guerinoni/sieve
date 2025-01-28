.SILENT : 

.PHONY: default
default:
	echo "Available targets:"
	perl -n -e 'print "- $$1\n" if (/^([a-z][a-z_0-9-]+)\s*:\s/ && $$1 ne "default")' $(MAKEFILE_LIST)

PKG := ./...
COVERAGE_FILE := coverage.out

# Platform-specific coverage tool handling
COVER_TOOL := $(shell command -v go || echo "")
COVER_VIEWER := $(shell command -v open || command -v xdg-open || echo "")

# Error if Go is not installed
ifeq ($(COVER_TOOL),)
$(error Go is not installed. Please install Go to use this Makefile.)
endif

.PHONY: test
test:
	echo Running tests...
	go test -race $(PKG) -v

.PHONY: test-escape
test-escape:
	echo Running tests with escape analysis...
	go test -gcflags="-m" $(PKG)

.PHONY: coverage
coverage:
	echo Running tests with coverage...
	go test -coverprofile=$(COVERAGE_FILE) $(PKG)
	echo Coverage details:
	go tool cover -func=$(COVERAGE_FILE)

.PHONY: coverage-html
coverage-html: coverage
	echo Generating HTML coverage report...
	go tool cover -html=$(COVERAGE_FILE)
	if [ -n "$(COVER_VIEWER)" ]; then \
		echo Opening HTML coverage report...; \
		$(COVER_VIEWER) $(COVERAGE_FILE); \
	else \
		echo No suitable application found to open the HTML coverage report.; \
	fi

.PHONY: bench
bench:
	echo Running benchmarks...
	go test -bench=. -benchtime=5s -benchmem $(PKG)

.PHONY: bench-cmp
bench-cmp:
	echo install benchstat...
	go install golang.org/x/perf/cmd/benchstat@latest
	go test -bench=. -benchtime=5s -benchmem $(PKG) > new.txt
	benchstat benches.txt new.txt

.PHONY: lint
lint:
	echo Running linter...
	golangci-lint run --fix ./...

.PHONY: clean
clean:
	echo Cleaning up...
	go clean
	rm -f $(COVERAGE_FILE)
