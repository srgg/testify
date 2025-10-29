//go:generate go run ./cmd/dependgen TestifyLifecycleSuite

package depend_test

import (
	"encoding/json"
	"testing"

	"github.com/srgg/testify/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// callRecord tracks a single method call with hierarchical nested calls
type callRecord struct {
	Name     string        `json:"name"`
	Children []*callRecord `json:"children,omitempty"`
}

// lifecycleTracker tracks hierarchical execution flow using enter/exit
type lifecycleTracker struct {
	root    *callRecord // Root record (parent is nil)
	current *callRecord // Current active record
	parent  *callRecord // Parent of current record
}

func newLifecycleTracker() *lifecycleTracker {
	root := &callRecord{Name: "root"}
	return &lifecycleTracker{
		root:    root,
		current: root,
		parent:  nil,
	}
}

// onEnter creates a new call record and descends into it
func (lt *lifecycleTracker) onEnter(name string) {
	record := &callRecord{Name: name}
	lt.current.Children = append(lt.current.Children, record)
	lt.parent = lt.current
	lt.current = record
}

// onExit ascends back to the parent record
func (lt *lifecycleTracker) onExit() {
	if lt.parent != nil {
		lt.current = lt.parent
		// Find the parent's parent by traversing from the root
		lt.parent = lt.findParent(lt.root, lt.current)
	}
}

// findParent finds the parent of the target record by traversing from the root
func (lt *lifecycleTracker) findParent(node, target *callRecord) *callRecord {
	if node == target {
		return nil // target is root or direct child of the root
	}
	for _, child := range node.Children {
		if child == target {
			return node
		}
		if parent := lt.findParent(child, target); parent != nil {
			return parent
		}
	}
	return nil
}

// toJSON serializes the call tree to JSON (returns children of root, not root itself)
func (lt *lifecycleTracker) toJSON() ([]byte, error) {
	return json.MarshalIndent(lt.root.Children, "", "  ")
}

// assertJSONEqual compares tracker JSON with expected JSON string
func assertJSONEqual(t *testing.T, tracker *lifecycleTracker, expectedJSON string) {
	t.Helper()
	actual, err := tracker.toJSON()
	assert.NoError(t, err, "Failed to marshal tracker to JSON")

	var actualObj, expectedObj []*callRecord
	assert.NoError(t, json.Unmarshal(actual, &actualObj), "Failed to unmarshal actual JSON")
	assert.NoError(t, json.Unmarshal([]byte(expectedJSON), &expectedObj), "Failed to unmarshal expected JSON")

	assert.Equal(t, expectedObj, actualObj, "Lifecycle call tracking mismatch")
}

// TestifyLifecycleSuite tests that testify/suite lifecycle methods are called correctly
type TestifyLifecycleSuite struct {
	suite.Suite
	tracker *lifecycleTracker
}

// SetupSuite is called ONCE before all tests
func (s *TestifyLifecycleSuite) SetupSuite() {
	s.tracker.onEnter("SetupSuite")
	defer s.tracker.onExit()
}

// TearDownSuite is called ONCE after all tests
func (s *TestifyLifecycleSuite) TearDownSuite() {
	s.tracker.onEnter("TearDownSuite")
	defer s.tracker.onExit()
}

// SetupTest is called before EACH test method
func (s *TestifyLifecycleSuite) SetupTest() {
	s.tracker.onEnter("SetupTest(" + s.T().Name() + ")")
	// Don't exit - TearDownTest will exit
}

// TearDownTest is called after EACH test method
func (s *TestifyLifecycleSuite) TearDownTest() {
	s.tracker.onExit() // Exit from SetupTest
}

// BeforeTest is called before EACH test method (with suite and test name)
func (s *TestifyLifecycleSuite) BeforeTest(_ string, testName string) {
	s.tracker.onEnter("BeforeTest(" + testName + ")")
	// Don't exit - AfterTest will exit
}

// AfterTest is called after EACH test method (with suite and test name)
func (s *TestifyLifecycleSuite) AfterTest(_ string, testName string) {
	s.tracker.onExit() // Exit from BeforeTest
}

// SetupSubTest is called before EACH subtest
func (s *TestifyLifecycleSuite) SetupSubTest() {
	s.tracker.onEnter("SetupSubTest(" + s.T().Name() + ")")
	// Don't exit - TearDownSubTest will exit
}

// TearDownSubTest is called after EACH subtest
func (s *TestifyLifecycleSuite) TearDownSubTest() {
	s.tracker.onExit() // Exit from SetupSubTest
}

func (s *TestifyLifecycleSuite) TestA() {
	s.tracker.onEnter("TestA")
	defer s.tracker.onExit()
}

// @dependsOn TestA
func (s *TestifyLifecycleSuite) TestB() {
	s.tracker.onEnter("TestB")
	defer s.tracker.onExit()
}

// @dependsOn TestB
func (s *TestifyLifecycleSuite) TestWithNestedSubtests() {
	s.tracker.onEnter("TestWithNestedSubtests")
	defer s.tracker.onExit()

	s.Run("Level1", func() {
		s.tracker.onEnter("Level1")
		defer s.tracker.onExit()

		s.Run("Level2A", func() {
			s.tracker.onEnter("Level2A")
			defer s.tracker.onExit()

			s.Run("Level3", func() {
				s.tracker.onEnter("Level3")
				defer s.tracker.onExit()
			})
		})
		s.Run("Level2B", func() {
			s.tracker.onEnter("Level2B")
			defer s.tracker.onExit()
		})
	})
}

func TestTestifyLifecycleSuite(t *testing.T) {
	// GOAL: Verify all  testify/suite lifecycle hooks are called correctly with dependency management
	//
	// TEtheST SCENARIO: Run suite with depend.RunSuite → lifecycle hooks called in hierarchical order → JSON matches an expected call tree

	s := &TestifyLifecycleSuite{tracker: newLifecycleTracker()}

	// Run the suite using depend.RunSuite (uses generated config)
	result := depend.RunSuite(t, s)

	// Verify tests passed
	assert.Equal(t, depend.TestPassed, result.Tests["TestA"].Status, "TestA should pass")
	assert.Equal(t, depend.TestPassed, result.Tests["TestB"].Status, "TestB should pass")
	assert.Equal(t, depend.TestPassed, result.Tests["TestWithNestedSubtests"].Status, "TestWithNestedSubtests should pass")

	// Verify all eight lifecycle hooks called correctly with nested subtests
	expectedJSON := `[
		{"name": "SetupSuite"},
		{
			"name": "BeforeTest(TestA)",
			"children": [
				{
					"name": "SetupTest(TestTestifyLifecycleSuite/TestA)",
					"children": [
						{"name": "TestA"}
					]
				}
			]
		},
		{
			"name": "BeforeTest(TestB)",
			"children": [
				{
					"name": "SetupTest(TestTestifyLifecycleSuite/TestB)",
					"children": [
						{"name": "TestB"}
					]
				}
			]
		},
		{
			"name": "BeforeTest(TestWithNestedSubtests)",
			"children": [
				{
					"name": "SetupTest(TestTestifyLifecycleSuite/TestWithNestedSubtests)",
					"children": [
						{
							"name": "TestWithNestedSubtests",
							"children": [
								{
									"name": "SetupSubTest(TestTestifyLifecycleSuite/TestWithNestedSubtests/Level1)",
									"children": [
										{
											"name": "Level1",
											"children": [
												{
													"name": "SetupSubTest(TestTestifyLifecycleSuite/TestWithNestedSubtests/Level1/Level2A)",
													"children": [
														{
															"name": "Level2A",
															"children": [
																{
																	"name": "SetupSubTest(TestTestifyLifecycleSuite/TestWithNestedSubtests/Level1/Level2A/Level3)",
																	"children": [
																		{"name": "Level3"}
																	]
																}
															]
														}
													]
												},
												{
													"name": "SetupSubTest(TestTestifyLifecycleSuite/TestWithNestedSubtests/Level1/Level2B)",
													"children": [
														{"name": "Level2B"}
													]
												}
											]
										}
									]
								}
							]
						}
					]
				}
			]
		},
		{"name": "TearDownSuite"}
	]`
	assertJSONEqual(t, s.tracker, expectedJSON)
}

// TContextTestSuite is a test suite for verifying the correct T context
type TContextTestSuite struct {
	suite.Suite
	setupSuiteName    string
	tearDownSuiteName string
	lastTestName      string
}

func (s *TContextTestSuite) SetupSuite() {
	s.setupSuiteName = s.T().Name()
}

func (s *TContextTestSuite) TearDownSuite() {
	s.tearDownSuiteName = s.T().Name()
}

// TestTearDownSuiteReceivesCorrectT verifies that TearDownSuite() is called with the parent test's T,
// not the last subtest's T. This is a regression test for the issue where s.SetT(t) wasn't restored
// after the test loop completed.
func TestTearDownSuiteReceivesCorrectT(t *testing.T) {
	// GOAL: Verify TearDownSuite receives the parent test T, not the last subtest's T
	// TEST SCENARIO:
	// 1. Suite captures t.Name() in SetupSuite → should be the parent test name
	// 2. Suite runs multiple tests → each has its own T with different names
	// 3. Suite captures t.Name() in TearDownSuite → should match SetupSuite (parent test name)

	s := &TContextTestSuite{}

	// Create a minimal config with three tests to ensure we have multiple subtests
	reg := map[string]func(any){
		"TestFirst": func(suite any) {
			ts := suite.(*TContextTestSuite)
			ts.lastTestName = ts.T().Name()
		},
		"TestSecond": func(suite any) {
			ts := suite.(*TContextTestSuite)
			ts.lastTestName = ts.T().Name()
		},
		"TestThird": func(suite any) {
			ts := suite.(*TContextTestSuite)
			ts.lastTestName = ts.T().Name()
		},
	}
	order := []string{"TestFirst", "TestSecond", "TestThird"}
	dep := &depend.Dep{Deps: map[string][]string{}}

	// Run the suite
	depend.RunSuiteWithConfig(t, s, reg, order, dep)

	// Verify SetupSuite and TearDownSuite received the same T (parent test context)
	assert.Equal(t, s.setupSuiteName, s.tearDownSuiteName,
		"SetupSuite and TearDownSuite should receive the same T (parent test context)")

	// Verify TearDownSuite did NOT receive the last test's T
	assert.NotEqual(t, s.tearDownSuiteName, s.lastTestName,
		"TearDownSuite should NOT receive the last subtest's T")

	// Verify SetupSuite and TearDownSuite received the parent test's name
	assert.Equal(t, t.Name(), s.setupSuiteName,
		"SetupSuite should receive parent test's T")
	assert.Equal(t, t.Name(), s.tearDownSuiteName,
		"TearDownSuite should receive parent test's T")
}
