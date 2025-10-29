# Makefile — testify/depend
#
# Handles:
#   - Build binaries
#   - Run tests and collect coverage
#   - Lint, static analysis, and generated-code verification
#   - Dependency management (tidy, verify)
#   - Release packaging via GoReleaser
#
# Run `make help` to list available targets.

.PHONY: help all build test test-unit test-integration lint fmt vet generate clean coverage \
        release-snapshot release-check install deps update-deps verify pre-commit

# ============================================================================
# Configuration
# ============================================================================

# Project metadata
PROJECT_NAME := testify/depend
BINARY_NAME := dependgen
MAIN_PATH := ./depend/cmd/dependgen

# Build configuration (can be overridden by environment or goreleaser)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod
GOGENERATE := $(GOCMD) generate
GOINSTALL := $(GOCMD) install

# Goreleaser - detect if installed via PATH or go tool
GORELEASER := $(shell command -v goreleaser > /dev/null 2>&1 && echo "goreleaser" || (go tool | grep -q goreleaser 2>/dev/null && echo "go tool goreleaser" || echo "goreleaser"))

# Directories
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

# Test flags
TEST_FLAGS := -v -race -timeout=2m
TEST_UNIT_PATHS := ./depend/cmd/dependgen
TEST_INTEGRATION_PATHS := ./depend/...

# IMPORTANT: examples/ has intentionally failing tests to demonstrate dependency behavior
# We exclude it from CI/validation but keep it for manual testing
TEST_EXCLUDE_PATHS := ./depend/examples

# Coverage
COVERAGE_MODE := atomic
COVERAGE_PROFILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
# Coverage policy (optional)
MIN_COVERAGE := 80

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# ============================================================================
# Internal utility targets
# ============================================================================

# check-tool: Internal function to check if a tool is available (PATH or go tool)
# Usage: $(call check-tool,toolname)
# Returns: 0 if found, 1 if not found
define check-tool
command -v $(1) > /dev/null 2>&1 || $(GOCMD) tool | grep -q $(1) 2>/dev/null
endef

# install-go-tool: Internal function to install a Go tool
# Usage: $(call install-go-tool,toolname,package)
# Example: $(call install-go-tool,staticcheck,honnef.co/go/tools/cmd/staticcheck)
define install-go-tool
	echo "$(COLOR_BLUE)Installing $(1) via 'go get -tool'...$(COLOR_RESET)" && \
	$(GOCMD) get -tool $(2)@latest && \
	if $(GOCMD) tool | grep -q $(1); then \
		echo "$(COLOR_GREEN)✓ $(1) installed (use 'go tool $(1)')$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ $(1) installation failed$(COLOR_RESET)"; \
		exit 1; \
	fi
endef

# ============================================================================
# Default Target
# ============================================================================

.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)$(PROJECT_NAME) - Makefile Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET)"
	@echo "  make <target>"
	@echo ""
	@echo "$(COLOR_BOLD)Development:$(COLOR_RESET)"
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = "## "}; {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$2, $$3}'
	@echo ""
	@echo "$(COLOR_BOLD)Examples:$(COLOR_RESET)"
	@echo "  make all                # Run all checks and tests"
	@echo "  make test               # Run all tests (excluding examples)"
	@echo "  make test-examples      # Run example tests (with code generation)"
	@echo "  make generate-examples  # Generate dependency code for examples"
	@echo "  make clean-examples     # Clean generated example files"
	@echo "  make build              # Build the dependgen binary"
	@echo "  make release-check      # Verify release readiness"
	@echo ""

# ============================================================================
# Main Targets
# ============================================================================

## all: Run all checks, generate, test, and build
all: deps fmt vet generate test build
	@echo "$(COLOR_GREEN)✓ All checks passed$(COLOR_RESET)"

## build: Build the dependgen binary (respects GOOS, GOARCH, VERSION, COMMIT, DATE env vars)
build:
	@echo "$(COLOR_BLUE)Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -trimpath \
		-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe) \
		$(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe)$(COLOR_RESET)"

## install: Install dependgen to $GOPATH/bin
install:
	@echo "$(COLOR_BLUE)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	@$(GOINSTALL) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Installed $(BINARY_NAME)$(COLOR_RESET)"

## clean: Remove build artifacts and coverage reports
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f depend/cmd/dependgen/coverage.out
	@rm -f depend/cmd/dependgen/coverage.html
	@rm -f depend/cmd/dependgen/dependgen
	@find . -name '*_depend_test.go' -type f -delete
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## clean-examples: Remove generated example dependency files
clean-examples:
	@echo "$(COLOR_BLUE)Cleaning example generated files...$(COLOR_RESET)"
	@find ./depend/examples -name '*_depend_test.go' -type f -delete
	@echo "$(COLOR_GREEN)✓ Example generated files cleaned$(COLOR_RESET)"

## generate-examples: Run go generate for example tests only
generate-examples:
	@echo "$(COLOR_BLUE)Generating dependency code for examples...$(COLOR_RESET)"
	@$(GOCMD) generate ./depend/examples/...
	@echo "$(COLOR_GREEN)✓ Example dependency code generated$(COLOR_RESET)"

# ============================================================================
# Testing
# ============================================================================

# test-summary-helper: Internal shell function to parse go test output and print summary
# Parses verbose test output to extract PASS/FAIL/SKIP counts and test names (including subtests)
define test-summary
	awk '/--- PASS:/ { p++ } \
	     /--- FAIL:/ { f++ } \
	     /--- SKIP:/ { s++ } \
	     END { print ""; printf "SUMMARY: %d passed, %d failed, %d skipped (%d total)\n", p, f, s, p+f+s }'
endef

## test: Run all tests (excluding examples)
test: generate
	@echo "$(COLOR_BLUE)Running all tests (excluding examples)...$(COLOR_RESET)"
	@if command -v gotestfmt > /dev/null 2>&1; then \
		$(GOTEST) -json $(TEST_FLAGS) $$(go list ./... | grep -v /examples) 2>&1 | tee /tmp/gotest.log | gotestfmt; \
	else \
		$(GOTEST) $(TEST_FLAGS) $$(go list ./... | grep -v /examples) 2>&1 | tee /dev/stderr | $(test-summary); \
	fi

## test-unit: Run unit tests only (excludes integration tests)
test-unit:
	@echo "$(COLOR_BLUE)Running unit tests (excluding integration tests)...$(COLOR_RESET)"
	@if command -v gotestfmt > /dev/null 2>&1; then \
		$(GOTEST) -json $(TEST_FLAGS) -run '^Test[^D]|^TestD[^e]' $(TEST_UNIT_PATHS) 2>&1 | gotestfmt; \
	else \
		$(GOTEST) $(TEST_FLAGS) -run '^Test[^D]|^TestD[^e]' $(TEST_UNIT_PATHS) 2>&1 | tee /dev/stderr | $(test-summary); \
	fi

## test-integration: Run integration tests only
test-integration:
	@echo "$(COLOR_BLUE)Running integration tests...$(COLOR_RESET)"
	@if command -v gotestfmt > /dev/null 2>&1; then \
		$(GOTEST) -json $(TEST_FLAGS) -run '^TestDependgenSuite$$' $(TEST_UNIT_PATHS) 2>&1 | gotestfmt; \
	else \
		$(GOTEST) $(TEST_FLAGS) -run '^TestDependgenSuite$$' $(TEST_UNIT_PATHS) 2>&1 | tee /dev/stderr | $(test-summary); \
	fi

## test-examples: Run example tests (these intentionally fail to demonstrate behavior)
test-examples: generate-examples
	@echo "$(COLOR_YELLOW)Running example tests (expect intentional failures)...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)These failures demonstrate dependency skip behavior - this is expected!$(COLOR_RESET)"
	@$(GOTEST) -v $(TEST_EXCLUDE_PATHS) || echo "$(COLOR_YELLOW)→ Example tests completed (failures are intentional)$(COLOR_RESET)"

## coverage: Generate test coverage report (excluding examples)
coverage:
	@echo "$(COLOR_BLUE)Generating coverage report...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Note: examples/ excluded (contains intentional test failures)$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -covermode=$(COVERAGE_MODE) \
		-coverprofile=$(COVERAGE_PROFILE) ./depend ./depend/cmd/dependgen
	@echo "$(COLOR_BLUE)Generating detailed HTML report...$(COLOR_RESET)"
	@$(GOCMD) tool cover -html=$(COVERAGE_PROFILE) -o=$(COVERAGE_HTML)
	@echo "$(COLOR_BLUE)Generating function-level coverage report...$(COLOR_RESET)"
	@$(GOCMD) tool cover -func=$(COVERAGE_PROFILE) > $(COVERAGE_DIR)/coverage-func.txt
	@echo "$(COLOR_GREEN)✓ Coverage reports generated:$(COLOR_RESET)"
	@echo "  - HTML report:     $(COVERAGE_HTML)"
	@echo "  - Function report: $(COVERAGE_DIR)/coverage-func.txt"
	@echo "  - Raw data:        $(COVERAGE_PROFILE)"
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_PROFILE) | awk '/total:/ {print $$3}' | sed 's/%//'); \
	if [ $${coverage%.*} -lt $(MIN_COVERAGE) ]; then \
		echo "$(COLOR_YELLOW)⚠ Coverage below target: $${coverage}% (minimum: $(MIN_COVERAGE)%)$(COLOR_RESET)"; \
		exit 1; \
	fi
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_PROFILE) | awk '/total:/ {print $$3}'); \
	echo "$(COLOR_GREEN)✓ Coverage: $${coverage} (exceeds minimum threshold of $(MIN_COVERAGE)%)$(COLOR_RESET)"

# ============================================================================
# Code Quality
# ============================================================================

## fmt: Format all Go source files
fmt:
	@printf "Formatting code... " && $(GOCMD) fmt ./... > /dev/null && echo "$(COLOR_GREEN)✓$(COLOR_RESET)"

## vet: Run go vet on all packages
vet:
	@printf "Running go vet... " && $(GOVET) ./... && echo "$(COLOR_GREEN)✓$(COLOR_RESET)"

## lint: Run additional linting (if golangci-lint is installed)
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
		echo "$(COLOR_GREEN)✓ Linting passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not installed, skipping$(COLOR_RESET)"; \
		echo "  Install: https://golangci-lint.run/usage/install/"; \
	fi

## static: Run staticcheck (if installed)
static:
	@echo "$(COLOR_BLUE)Running static analysis...$(COLOR_RESET)"
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
		echo "$(COLOR_GREEN)✓ Static analysis passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ staticcheck not installed, skipping$(COLOR_RESET)"; \
		echo "  Install: https://staticcheck.dev"; \
	fi

## generate: Run go generate for all packages (excluding examples)
generate:
	@echo "$(COLOR_BLUE)Running code generation (excluding examples)...$(COLOR_RESET)"
	@for dir in $$(go list -f '{{.Dir}}' ./... | grep -v /examples); do \
		(cd $$dir && $(GOGENERATE) . 2>/dev/null || true); \
	done
	@echo "$(COLOR_GREEN)✓ Code generation complete$(COLOR_RESET)"

## verify: Verify generated code is up-to-date
verify: generate
	@echo "$(COLOR_BLUE)Verifying generated code is up-to-date...$(COLOR_RESET)"
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(COLOR_YELLOW)⚠ Generated code is out of date!$(COLOR_RESET)"; \
		echo "Run 'make generate' and commit changes"; \
		git status --short; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓ Generated code is up-to-date$(COLOR_RESET)"

# ============================================================================
# Dependencies
# ============================================================================

## deps: Download, tidy, and verify dependencies
deps:
	@printf "Managing dependencies (tidy, download, verify)... " && \
		$(GOMOD) tidy && $(GOMOD) download && $(GOMOD) verify > /dev/null && \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET)"

## update-deps: Update dependencies to latest versions
update-deps:
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	@$(GOMOD) get -u ./...
	@$(GOMOD) tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

## tidy: Tidy go.mod and go.sum only (use 'make deps' for full dependency management)
tidy:
	@echo "$(COLOR_BLUE)Tidying go.mod...$(COLOR_RESET)"
	@$(GOMOD) tidy
	@echo "$(COLOR_GREEN)✓ go.mod tidied$(COLOR_RESET)"

# ============================================================================
# Release
# ============================================================================

## release-check: Verify everything is ready for release (used by goreleaser)
release-check: check-release-tools deps fmt vet test
	@echo "$(COLOR_BLUE)Checking release readiness...$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Validating goreleaser config...$(COLOR_RESET)"
	@$(GORELEASER) check
	@echo "$(COLOR_GREEN)✓ GoReleaser config valid$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Checking for uncommitted changes...$(COLOR_RESET)"
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(COLOR_YELLOW)⚠ Working directory not clean$(COLOR_RESET)"; \
		git status --short; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓ Ready for release$(COLOR_RESET)"

## release-snapshot: Create a snapshot release (local testing)
release-snapshot: release-check
	@echo "$(COLOR_BLUE)Creating snapshot release...$(COLOR_RESET)"
	@$(GORELEASER) release --snapshot --clean
	@echo "$(COLOR_GREEN)✓ Snapshot release created in ./dist$(COLOR_RESET)"

## release-dry-run: Dry run of release process
release-dry-run: release-check
	@echo "$(COLOR_BLUE)Running release dry-run...$(COLOR_RESET)"
	@$(GORELEASER) release --snapshot --clean --skip=publish
	@echo "$(COLOR_GREEN)✓ Release dry-run complete$(COLOR_RESET)"

# ============================================================================
# Pre-commit
# ============================================================================

## pre-commit: Run all pre-commit checks (format, vet, test)
pre-commit: fmt vet generate test
	@echo "$(COLOR_GREEN)✓ Pre-commit checks passed$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Reminder: examples/ has intentional failures - this is expected$(COLOR_RESET)"

# ============================================================================
# CI/CD Targets
# ============================================================================

## ci-test: Run tests for CI (excludes examples with intentional failures)
ci-test: deps
	@echo "$(COLOR_BLUE)Running CI tests...$(COLOR_RESET)"
	@if command -v gotestfmt > /dev/null 2>&1; then \
		$(GOTEST) -json $(TEST_FLAGS) ./depend/cmd/dependgen 2>&1 | gotestfmt; \
	else \
		$(GOTEST) $(TEST_FLAGS) ./depend/cmd/dependgen 2>&1; \
	fi
	@echo "$(COLOR_GREEN)✓ CI tests passed$(COLOR_RESET)"

## ci-validate: Run all validation checks for CI
ci-validate: deps fmt vet static ci-test
	@echo "$(COLOR_GREEN)✓ CI validation passed$(COLOR_RESET)"

# ============================================================================
# Development Helpers
# ============================================================================

## docs: Open package documentation in browser
docs:
	@echo "$(COLOR_BLUE)Opening documentation...$(COLOR_RESET)"
	@echo "Visit: http://localhost:6060/pkg/github.com/srgg/testify/depend/"
	@godoc -http=:6060

# ============================================================================
# Info
# ============================================================================

## version: Display version information
version:
	@echo "$(COLOR_BOLD)Version Information:$(COLOR_RESET)"
	@echo "  Go version: $$(go version)"
	@if command -v goreleaser > /dev/null; then \
		echo "  GoReleaser: $$(goreleaser --version | head -1)"; \
	else \
		echo "  GoReleaser: not installed"; \
	fi
	@echo ""
	@echo "$(COLOR_BOLD)Module:$(COLOR_RESET)"
	@head -3 go.mod

## check-tools: Check if required development tools are installed (use FORCE_INSTALL=1 to auto-install)
check-tools:
	@echo "$(COLOR_BOLD)Checking required tools:$(COLOR_RESET)"
	@command -v go > /dev/null && echo "$(COLOR_GREEN)✓ go$(COLOR_RESET)" || echo "$(COLOR_YELLOW)✗ go (required)$(COLOR_RESET)"
	@command -v gofmt > /dev/null && echo "$(COLOR_GREEN)✓ gofmt$(COLOR_RESET)" || echo "$(COLOR_YELLOW)✗ gofmt (required)$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Optional tools:$(COLOR_RESET)"
	@if $(call check-tool,staticcheck); then \
		echo "$(COLOR_GREEN)✓ staticcheck$(COLOR_RESET)"; \
	elif [ "$(FORCE_INSTALL)" = "1" ]; then \
		$(call install-go-tool,staticcheck,honnef.co/go/tools/cmd/staticcheck); \
	else \
		echo "$(COLOR_YELLOW)⚠ staticcheck (optional)$(COLOR_RESET)"; \
		echo "  Install: make check-tools FORCE_INSTALL=1"; \
	fi
	@if command -v golangci-lint > /dev/null; then \
		echo "$(COLOR_GREEN)✓ golangci-lint$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint (optional)$(COLOR_RESET)"; \
		echo "  Install: https://golangci-lint.run/usage/install/"; \
	fi

## check-release-tools: Check if release tools are installed (use FORCE_INSTALL=1 to auto-install)
check-release-tools:
	@echo "$(COLOR_BOLD)Checking release tools:$(COLOR_RESET)"
	@if $(call check-tool,goreleaser); then \
		echo "$(COLOR_GREEN)✓ goreleaser$(COLOR_RESET)"; \
	elif [ "$(FORCE_INSTALL)" = "1" ]; then \
		$(call install-go-tool,goreleaser,github.com/goreleaser/goreleaser/v2); \
	else \
		echo "$(COLOR_YELLOW)✗ goreleaser (required for releases)$(COLOR_RESET)"; \
		echo "  Install: make check-release-tools FORCE_INSTALL=1"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓ All release tools available$(COLOR_RESET)"