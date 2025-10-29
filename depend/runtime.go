package depend

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestStatus represents the execution status of a test or suite.
type TestStatus int

const (
	// TestNotRun indicates the test has not been executed yet.
	TestNotRun TestStatus = iota
	// TestPassed indicates the test executed successfully.
	TestPassed
	// TestFailed indicates the test executed but failed.
	TestFailed
	// TestSkipped indicates the test was skipped (usually due to dependency failure).
	TestSkipped
)

// String returns a human-readable representation of the test status.
func (s TestStatus) String() string {
	switch s {
	case TestNotRun:
		return "NotRun"
	case TestPassed:
		return "Passed"
	case TestFailed:
		return "Failed"
	case TestSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// TestResult represents the result of a single test.
type TestResult struct {
	Name   string     // Test name (e.g., "TestLogin")
	Status TestStatus // Execution status
}

// SuiteResult represents the complete result of a test suite.
type SuiteResult struct {
	SuiteName string                 // Suite name (e.g., "AuthSuite")
	Status    TestStatus             // Overall suite status
	Tests     map[string]*TestResult // Top-level test results
}

// TestingT extends testing.TB with the Run method.
// This interface allows accepting both *testing.T (used in production) and test doubles
// (like SpyT in runtime_test.go) that need to intercept test execution for verification.
// testing.TB doesn't include Run(), but we need it to create subtests for each test in the suite.
type TestingT interface {
	testing.TB
	Run(name string, f func(t *testing.T)) bool
}

// SuiteConfig packages metadata for running test suites with dependencies.
// Registry maps test names to their functions.
// Order defines an execution sequence.
// Deps contains parent-child dependency relationships.
type SuiteConfig struct {
	Registry map[string]func(any)
	Order    []string
	Deps     *Dep
}

// Configurable must be implemented by test suites using dependency management.
// GeneratedDependConfig returns suite-specific configuration produced by code generation.
type Configurable interface {
	suite.TestingSuite
	GeneratedDependConfig() *SuiteConfig
}

// recoverAndFailOnPanic recovers from panics in test lifecycle methods and fails the test.
// This matches the behavior of testify/suite to ensure panics are properly reported.
func recoverAndFailOnPanic(t TestingT) {
	t.Helper()
	if r := recover(); r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}

// RunSuite executes suite tests respecting declared dependencies.
// Suite must implement the Configurable interface via code generation.
// Tests skip automatically if the required parent tests fail.
// Returns complete suite result with test statuses and hierarchical structure.
func RunSuite(t TestingT, s suite.TestingSuite) *SuiteResult {
	// Check if suite provides dependency configuration
	configurable, ok := s.(Configurable)
	if !ok {
		// Intentional panic following Go best practices for code generation tools.
		// This immediately alerts developers about missing generated code with actionable steps.
		// Similar pattern used by: go:embed (missing files), protobuf (missing .pb.go), and
		// mockery (missing mocks).
		panic(fmt.Sprintf("testify/depend: Suite %T is missing the GeneratedDependConfig() method (code generation required).\n\n"+
			"Run the dependgen code generator:\n"+
			"  go get -tool github.com/srgg/testify/depend/cmd/dependgen  # Install (Go 1.24+)\n"+
			"  //go:generate go tool github.com/srgg/testify/depend/cmd/dependgen %T  # Add to test file\n"+
			"  go generate ./...  # Generate code\n\n"+
			"For Go < 1.24: go install github.com/srgg/testify/depend/cmd/dependgen@latest",
			s, s))
	}

	config := configurable.GeneratedDependConfig()
	return RunSuiteWithConfig(t, s, config.Registry, config.Order, config.Deps)
}

// RunSuiteWithConfig executes suite tests with explicit dependency configuration.
// Prefer RunSuite for code-generated configuration.
// Tests execute in a specified order, skipping if parent dependencies fail.
// Returns complete suite result with test statuses and hierarchical structure.
func RunSuiteWithConfig(
	t TestingT,
	s suite.TestingSuite,
	reg map[string]func(any),
	order []string,
	dep *Dep,
) *SuiteResult {
	// Recover from panics in the overall suite execution
	defer recoverAndFailOnPanic(t)

	// Get suite name for lifecycle hooks (matches testify/suite behavior)
	methodFinder := reflect.TypeOf(s)
	suiteName := methodFinder.Elem().Name()

	// Build SuiteResult directly as tests execute
	suiteResult := &SuiteResult{
		SuiteName: suiteName,
		Tests:     make(map[string]*TestResult),
		Status:    TestPassed, // Will be updated if any test fails
	}

	// Extract *testing.T from TestingT (handles both *testing.T and wrappers like SpyT)
	var realT *testing.T
	if concreteT, ok := t.(*testing.T); ok {
		realT = concreteT
	} else if unwrapper, ok := t.(interface{ Unwrap() *testing.T }); ok {
		realT = unwrapper.Unwrap()
	} else {
		panic("TestingT must be *testing.T or provide Unwrap() *testing.T")
	}

	// Set T for the suite (required for SetupSuite to use t.Cleanup, etc.)
	s.SetT(realT)

	// Set parent suite reference for subtest lifecycle hooks (SetupSubTest/TearDownSubTest)
	s.SetS(s)

	// Call SetupSuite if the suite implements it (runs ONCE before all tests)
	if setupSuite, ok := s.(interface{ SetupSuite() }); ok {
		setupSuite.SetupSuite()
	}

	for _, name := range order {
		testFn := reg[name]
		parents := dep.Deps[name]

		// Check if any required parent failed
		shouldSkip := false
		var failedParent string
		for _, p := range parents {
			parentTest := suiteResult.Tests[p]
			if parentTest == nil || parentTest.Status != TestPassed {
				shouldSkip = true
				failedParent = p
				break
			}
		}

		if shouldSkip {
			t.Run(name, func(t *testing.T) {
				t.Skipf("Skipping %s: dependency %s failed", name, failedParent)
			})
			suiteResult.Tests[name] = &TestResult{
				Name:   name,
				Status: TestSkipped,
			}
			suiteResult.Status = TestFailed // Suite fails if any test is skipped
			continue
		}

		// Run the test and capture pass/fail status
		// t.Run() returns false if the test failed
		testPassed := t.Run(name, func(subT *testing.T) {
			// Recover from panics in test execution
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("test panicked: %v\n%s", r, debug.Stack())
					subT.Fail() // Mark the subtest as failed for proper test output
				}
			}()

			// Set T for this test
			s.SetT(subT)

			// Call BeforeTest if the suite implements it
			if beforeTest, ok := s.(interface {
				BeforeTest(suiteName, testName string)
			}); ok {
				beforeTest.BeforeTest(suiteName, name)
			}

			// Call SetupTest if the suite implements it
			if setupTest, ok := s.(interface{ SetupTest() }); ok {
				setupTest.SetupTest()
			}

			// Run the actual test
			testFn(s)

			// Call TearDownTest if the suite implements it
			if tearDownTest, ok := s.(interface{ TearDownTest() }); ok {
				tearDownTest.TearDownTest()
			}

			// Call AfterTest if the suite implements it
			if afterTest, ok := s.(interface {
				AfterTest(suiteName, testName string)
			}); ok {
				afterTest.AfterTest(suiteName, name)
			}
		})

		// Record test result after t.Run() completes
		testStatus := TestPassed
		if !testPassed {
			testStatus = TestFailed
			suiteResult.Status = TestFailed // Mark suite as failed
		}
		suiteResult.Tests[name] = &TestResult{
			Name:   name,
			Status: testStatus,
		}
	}

	// Restore parent test T for TearDownSuite
	s.SetT(realT)

	// Call TearDownSuite if the suite implements it (runs ONCE after all tests)
	if tearDownSuite, ok := s.(interface{ TearDownSuite() }); ok {
		tearDownSuite.TearDownSuite()
	}

	//// Record in global registry for cross-suite dependencies
	//global.RecordSuite(suiteResult)
	return suiteResult
}
