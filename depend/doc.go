// Package depend adds deterministic test-dependency management to
// github.com/stretchr/testify/suite test suites.
//
// Real-world integration tests often build on previous results
// (e.g., authentication before resource creation). When a prerequisite
// test fails, running downstream tests only adds noise. Depend solves this:
//
//   - Declare dependencies in code comments
//   - Generate dependency config (go:generate)
//   - Skip dependent tests automatically on failure
//   - Run tests strictly in source-declaration order
//
// All of this comes with zero overhead when tests pass.
//
// # Overview
//
// Depend augments testify/suite by enforcing explicit ordering and
// dependency rules between test methods. Each test:
//
//   - Runs only when all declared prerequisites are met
//   - Is clearly reported if skipped due to failed dependencies
//
// Dependencies are declared using @dependsOn directives placed directly
// above test methods. A lightweight code generator produces the suite
// execution plan and wiring.
//
// # Quick Start
//
// 1️⃣ Annotate tests with dependencies:
//
//	// @dependsOn TestLogin
//	func (s *APISuite) TestCreateUser() { … }
//
//	// @dependsOn TestCreateUser
//	func (s *APISuite) TestDeleteUser() { … }
//
// 2️⃣ Add a generation directive to the test file:
//
//	//go:generate go run github.com/srgg/testify/depend/cmd/dependgen
//
// 3️⃣ Generate code:
//
//	go generate ./...
//
// 4️⃣ Replace suite.Run with:
//
//	depend.RunSuite(t, new(APISuite))
//
// # Dependency Syntax
//
//	// Single
//	// @dependsOn TestA
//
//	// Multiple (comma-separated or repeated comments)
//	// @dependsOn TestA, TestB
//	// @dependsOn TestC
//
// Rules:
//
//   - Tests run in file declaration order
//   - Referenced tests must exist in the same suite
//   - Circular/self-dependencies are rejected at generation time
//
// # Execution Model
//
// When using RunSuite:
//
//  1. Tests are executed in discovered order
//  2. Pre-checks ensure dependencies passed
//  3. Failures skip dependents with clear messaging
//
// Example:
//
//	--- FAIL: TestLogin
//	--- SKIP: TestCreateUser (failed dependency: TestLogin)
//	--- SKIP: TestDeleteUser (failed dependency: TestCreateUser)
//
// # Code Generation
//
// dependgen produces a small, static config per suite:
//
//   - Registry: maps test names → methods
//   - Order: declaration order
//   - Deps: dependency graph
//
// The suite implements:
//
//	func (s *MySuite) GeneratedDependConfig() *depend.SuiteConfig
//
// Missing or stale generated code results in actionable errors.
//
// # Installation
//
// Recommended (Go 1.24+ tool management):
//
//	go get -tool github.com/srgg/testify/depend/cmd/dependgen@latest
//
// Older Go versions:
//
//	go install github.com/srgg/testify/depend/cmd/dependgen@latest
//
// # CI Integration
//
// Ensure generated artifacts are current:
//
//	go generate ./...
//	git diff --exit-code
//
// # Compatibility & Performance
//
//   - Go 1.18+ (any parameter support required)
//   - Go 1.24+ recommended for tool directive support (go get -tool)
//   - Works with testify/suite v1.3.0+ and assert/require
//   - Zero added cost unless failures occur
//   - Failures skip dependents immediately, reducing wasted time
//
// # Troubleshooting
//
//   - Tests still run independently? → Ensure depend.RunSuite is used.
//   - Panic: missing GeneratedDependConfig? → `go generate ./...`
//   - Skips unexpected? → Check dependency annotations
package depend
