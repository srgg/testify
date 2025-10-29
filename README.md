# testify/depend - circuit-breaker for test execution<br/><sup style="font-size: 0.2em; font-style: italic;">&nbsp;skip dependent tests when prerequisites fail</sup>

[![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)](https://go.dev/dl/)
[![Go Report Card](https://goreportcard.com/badge/github.com/srgg/testify)](https://goreportcard.com/report/github.com/srgg/testify)
![coverage](https://raw.githubusercontent.com/wiki/srgg/testify/coverage.svg)
[![Release](https://img.shields.io/github/v/release/srgg/testify)](https://github.com/srgg/testify/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
    

## Quick Example

```go
//go:generate go run github.com/srgg/testify/depend/cmd/dependgen
package examples

type SimpleSuite struct {
    suite.Suite
}

func (s *SimpleSuite) TestA() {
    // First test - no dependencies
}

// @dependsOn TestA
func (s *SimpleSuite) TestB() {
    // This test depends on TestA
}

// @dependsOn TestA, TestB
func (s *SimpleSuite) TestC() {
    // This test depends on BOTH TestA and TestB
}

func TestSimpleSuite(t *testing.T) {
    depend.RunSuite(t, new(SimpleSuite))
}
```

**When all tests pass:**
```
--- PASS: TestSimpleSuite/All_pass_-_no_failures (0.00s)
    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestA (0.00s)
    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestB (0.00s)
    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestC (0.00s)
```

**When TestA fails:**
```
--- FAIL: TestSimpleSuite/A_fails_-_B_and_C_skipped (0.00s)
    --- FAIL: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestA (0.00s)
    --- SKIP: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestB (0.00s)
    --- SKIP: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestC (0.00s)
```

**When TestB fails (but TestA passes):**
```
--- FAIL: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped (0.00s)
    --- PASS: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestA (0.00s)
    --- FAIL: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestB (0.00s)
    --- SKIP: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestC (0.00s)
```

This demonstrates:
- **Single dependencies** (TestB → TestA)
- **Multiple dependencies** (TestC → TestA, TestB)
- **Cascading skips** (when A fails, both B and C skip)
- **Targeted skips** (when B fails, only C skips)
- **Table-driven testing** with dependency management

## Installation

```bash
# Add the library
go get github.com/srgg/testify/depend

# Install the code generator
go install github.com/srgg/testify/depend/cmd/dependgen@latest
```

## How It Works

1. **Add `@dependsOn` comments** above test methods that have dependencies
2. **Run code generation** - `go generate` creates the dependency configuration
3. **Run your tests** - `depend.RunSuite()` handles dependency tracking automatically

Tests run in declaration order, automatically skipping when dependencies fail.

## Usage

### Step 1: Define Your Test Suite

Create a standard testify suite with a `//go:generate` directive:

```go
package mypackage_test

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/srgg/testify/depend"
)

//go:generate go run github.com/srgg/testify/depend/cmd/dependgen

type MySuite struct {
    suite.Suite
    // Your test data here
}
```

### Step 2: Write Tests with Dependencies

Add `@dependsOn` comments above tests that have dependencies:

```go
func (s *MySuite) TestSetup() {
    // Setup code
}

// @dependsOn TestSetup
func (s *MySuite) TestProcess() {
    // This test depends on TestSetup
}

// @dependsOn TestSetup, TestProcess
func (s *MySuite) TestValidate() {
    // This test depends on BOTH TestSetup and TestProcess
}
```

**Important:** Reference the full test method name, including the `Test` prefix.

### Step 3: Generate Dependency Configuration

```bash
go generate ./...
```

This generates dependency code in `<testfile>_depend_test.go` (e.g., `simple_test.go` → `simple_depend_test.go`).

**Note:** If you have multiple test suites in the same file, they will all be generated into a single output file based on the source filename. For example:
- `my_test.go` contains `FirstSuite` and `SecondSuite`
- Both generate into: `my_depend_test.go` (one file, two suite configurations)

### Step 4: Use depend.RunSuite

Replace `suite.Run` with `depend.RunSuite`:

```go
func TestMySuite(t *testing.T) {
    depend.RunSuite(t, new(MySuite))
}
```

### Step 5: Run Tests

```bash
go test -v
```

## Features

### Multiple Dependencies

A test can depend on multiple other tests. All must pass for the test to run:

```go
// @dependsOn TestDatabaseConnection, TestUserAuthentication
func (s *Suite) TestTransaction() {
    // Runs only if BOTH dependencies passed
}
```

### Cascading Skips

When a test fails, all tests that (directly or indirectly) depend on it are automatically skipped with clear messages:

```
--- SKIP: .../TestC (0.00s)
    depend: Skipping TestC: dependency TestA failed
```

### Declaration Order

Tests run in the order they're declared in your source file, making the flow easy to follow.

## Examples

### Complete Working Example

See [`depend/examples/simple_test.go`](depend/examples/simple_test.go) for a complete, runnable example demonstrating:
- Single dependencies
- Multiple dependencies
- Table-driven tests verifying skip behavior
- Proper usage of `@dependsOn` comments

To run the example:

```bash
cd depend/examples
go generate
go test -v
```

## Known Limitations

**Cross-suite dependencies are not yet supported.** The current implementation manages dependencies **within a single test suite only**. Tests in different suites cannot depend on each other.

A DAG-based cross-suite dependency runner is currently in development and will be available in a future release.

## Design Philosophy

### Why Comments Instead of Code?

Go lacks native metadata features like [Java annotations](https://docs.oracle.com/javase/tutorial/java/annotations/) or [TypeScript decorators](https://www.typescriptlang.org/docs/handbook/decorators.html). Comments are the cleanest way to express metadata without reflection or build hacks.

They keep things **simple, decoupled, and tooling-friendly** — how Go intends:

```go
// With comments - declaration lives with the test
// @dependsOn TestUserCreation
func (s *Suite) TestUserAuth() { /* ... */ }

// vs. separate configuration - easy to forget updating
var Dependencies = depend.Depends(...)
```

## Contributing

Contributions are welcome! Please open an issue before starting major work.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [stretchr/testify](https://github.com/stretchr/testify), the excellent testing toolkit for Go.
- Development assisted by [Claude Code](https://claude.com/claude-code) and [Serena](https://github.com/aorwall/serena) MCP server for semantic code operations.
- Released with [GoReleaser](https://goreleaser.com/) for multi-platform distribution.

---

**Questions?** Open an issue or start a discussion.

**Found a bug?** Please report it with a minimal reproduction case.