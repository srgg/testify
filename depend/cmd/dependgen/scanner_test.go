package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ScannerSuite struct {
	suite.Suite
}

func (s *ScannerSuite) TestUsesRunSuite() {
	// GOAL: Verify usesRunSuite correctly detects depend.RunSuite usage in AST
	//
	// TEST SCENARIO: Parse file with/without RunSuite → Check detection → Correct boolean returned

	type testCase struct {
		name           string
		code           string
		expectedResult bool
	}

	testCases := []testCase{
		{
			name: "File uses depend.RunSuite returns true",
			code: `package test
import "github.com/srgg/testify/depend"

func TestExample(t *testing.T) {
	depend.RunSuite(t, new(TestSuite))
}`,
			expectedResult: true,
		},
		{
			name: "File without RunSuite returns false",
			code: `package test

func TestExample(t *testing.T) {
	// No depend.RunSuite here
}`,
			expectedResult: false,
		},
		{
			name: "File uses other package RunSuite returns false",
			code: `package test
import "github.com/other/pkg"

func TestExample(t *testing.T) {
	other.RunSuite(t, new(TestSuite))
}`,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.code, parser.ParseComments)
			s.Require().NoError(err, "MUST parse valid Go code")

			scanner := NewSuiteScanner(&Config{})
			result := scanner.usesRunSuite(f)

			// All assertions MUST execute
			s.Assert().Equal(tc.expectedResult, result, "usesRunSuite result mismatch")
		})
	}
}

func (s *ScannerSuite) TestScanTestFiles() {
	// GOAL: Verify scanTestFiles correctly finds all test files in a directory
	//
	// TEST SCENARIO: Create directory with test files → scanTestFiles called → All test files found

	type testCase struct {
		name              string
		files             map[string]string // filename -> content
		expectedTestFiles []string          // expected test file names (relative)
		expectedCount     int
	}

	testCases := []testCase{
		{
			name: "Finds all test files",
			files: map[string]string{
				"a_test.go": "package test",
				"b_test.go": "package test",
				"main.go":   "package main",
			},
			expectedTestFiles: []string{"a_test.go", "b_test.go"},
			expectedCount:     2,
		},
		{
			name:              "Empty directory returns empty slice",
			files:             map[string]string{},
			expectedTestFiles: []string{},
			expectedCount:     0,
		},
		{
			name: "Only non-test files returns empty slice",
			files: map[string]string{
				"main.go":   "package main",
				"helper.go": "package helper",
				"util.go":   "package util",
			},
			expectedTestFiles: []string{},
			expectedCount:     0,
		},
		{
			name: "Mixed test and non-test files",
			files: map[string]string{
				"setup_test.go":    "package test",
				"helper.go":        "package helper",
				"teardown_test.go": "package test",
				"main.go":          "package main",
			},
			expectedTestFiles: []string{"setup_test.go", "teardown_test.go"},
			expectedCount:     2,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			fixture := NewTempDirFixture(s.T())
			for filename, content := range tc.files {
				fixture.WithFile(filename, content)
			}
			tmpDir := fixture.Build()

			files, err := scanTestFiles(tmpDir)

			// All assertions MUST execute
			s.Assert().NoError(err, "MUST scan without error")
			s.Assert().Len(files, tc.expectedCount, "MUST find expected number of test files")

			for _, expectedFile := range tc.expectedTestFiles {
				expectedPath := filepath.Join(tmpDir, expectedFile)
				s.Assert().Contains(files, expectedPath, "MUST include: %s", expectedFile)
			}
		})
	}
}

func (s *ScannerSuite) TestParseDependsOnComment() {
	// GOAL: Verify parseDependsOnComment correctly handles all comment formatting variations
	//
	// TEST SCENARIO: Parse various @dependsOn formats → Extract dependencies → Correct list returned

	scenarios := []struct {
		name         string
		comment      string
		expectedDeps []string
	}{
		{
			name:         "Standard comma-separated dependencies",
			comment:      "// @dependsOn TestA, TestB",
			expectedDeps: []string{"TestA", "TestB"},
		},
		{
			name:         "Whitespace variations",
			comment:      "//   @dependsOn   TestA  ,   TestB  ,TestC",
			expectedDeps: []string{"TestA", "TestB", "TestC"},
		},
		{
			name:         "Empty entries filtered out",
			comment:      "// @dependsOn TestA,,TestB",
			expectedDeps: []string{"TestA", "TestB"},
		},
		{
			name:         "Trailing comma ignored",
			comment:      "// @dependsOn TestA, TestB,",
			expectedDeps: []string{"TestA", "TestB"},
		},
		{
			name:         "Single dependency",
			comment:      "// @dependsOn TestA",
			expectedDeps: []string{"TestA"},
		},
		{
			name:         "No @dependsOn annotation",
			comment:      "// This is just a regular comment",
			expectedDeps: nil,
		},
		{
			name:         "Empty @dependsOn annotation",
			comment:      "// @dependsOn",
			expectedDeps: nil,
		},
		{
			name:         "Mixed whitespace and empty entries",
			comment:      "//  @dependsOn  TestA  , , TestB ,  ,TestC,",
			expectedDeps: []string{"TestA", "TestB", "TestC"},
		},
		{
			name:         "Case sensitivity preserved",
			comment:      "// @dependsOn TestSetUp, TestTearDown",
			expectedDeps: []string{"TestSetUp", "TestTearDown"},
		},
	}

	for _, tc := range scenarios {
		s.Run(tc.name, func() {
			deps := parseDependsOnComment(tc.comment)

			// All assertions MUST execute
			s.Assert().Equal(tc.expectedDeps, deps, "Dependencies mismatch for: %s", tc.comment)
		})
	}
}

func (s *ScannerSuite) TestScan() {
	// GOAL: Verify Scan() correctly processes multiple files and discovers suites
	//
	// TEST SCENARIO: Various file configurations → Scan() called → Correct suites and tests discovered

	type testCase struct {
		name           string
		files          map[string]string // filename -> content
		cfg            *Config
		expectError    bool
		errorContains  string
		expectedSuites []string
		validateTests  func(*ScannerSuite, []SuiteInfo)
	}

	testCases := []testCase{
		{
			name: "Single file with single suite",
			files: map[string]string{
				"test.go": `package mypackage

import "github.com/stretchr/testify/suite"
import "github.com/srgg/testify/depend"

type MySuite struct {
	suite.Suite
}

func (s *MySuite) TestA() {}
func (s *MySuite) TestB() {}

func TestRun(t *testing.T) {
	depend.RunSuite(t, new(MySuite))
}`,
			},
			cfg:            &Config{},
			expectError:    false,
			expectedSuites: []string{"MySuite"},
			validateTests: func(s *ScannerSuite, suiteInfo []SuiteInfo) {
				s.Assert().Len(suiteInfo, 1, "Should have 1 suite")
				s.Assert().Equal("mypackage", suiteInfo[0].Package, "Package should be mypackage")
				s.Assert().Len(suiteInfo[0].Tests, 2, "MySuite should have 2 tests")
				s.Assert().Equal("TestA", suiteInfo[0].Tests[0].Name)
				s.Assert().Equal("TestB", suiteInfo[0].Tests[1].Name)
			},
		},
		{
			name: "Multiple files with different suites",
			files: map[string]string{
				"first_test.go": `package mypackage

import "github.com/stretchr/testify/suite"
import "github.com/srgg/testify/depend"

type FirstSuite struct {
	suite.Suite
}

func (s *FirstSuite) TestA() {}

func TestFirst(t *testing.T) {
	depend.RunSuite(t, new(FirstSuite))
}`,
				"second_test.go": `package mypackage

import "github.com/stretchr/testify/suite"
import "github.com/srgg/testify/depend"

type SecondSuite struct {
	suite.Suite
}

func (s *SecondSuite) TestX() {}

func TestSecond(t *testing.T) {
	depend.RunSuite(t, new(SecondSuite))
}`,
			},
			cfg:            &Config{},
			expectError:    false,
			expectedSuites: []string{"FirstSuite", "SecondSuite"},
			validateTests: func(s *ScannerSuite, suiteInfo []SuiteInfo) {
				s.Assert().Len(suiteInfo, 2, "Should have 2 suites")
				// Find each suite and check test count
				for _, info := range suiteInfo {
					if info.Name == "FirstSuite" {
						s.Assert().Len(info.Tests, 1, "FirstSuite should have 1 test")
					} else if info.Name == "SecondSuite" {
						s.Assert().Len(info.Tests, 1, "SecondSuite should have 1 test")
					}
				}
			},
		},
		{
			name: "File without RunSuite is skipped",
			files: map[string]string{
				"with_runsuite.go": `package mypackage

import "github.com/stretchr/testify/suite"
import "github.com/srgg/testify/depend"

type ValidSuite struct {
	suite.Suite
}

func (s *ValidSuite) TestA() {}

func TestValid(t *testing.T) {
	depend.RunSuite(t, new(ValidSuite))
}`,
				"without_runsuite.go": `package mypackage

import "github.com/stretchr/testify/suite"

type SkippedSuite struct {
	suite.Suite
}

func (s *SkippedSuite) TestX() {}`,
			},
			cfg:            &Config{},
			expectError:    false,
			expectedSuites: []string{"ValidSuite"},
			validateTests: func(s *ScannerSuite, suiteInfo []SuiteInfo) {
				s.Assert().Len(suiteInfo, 1, "Only ValidSuite should be discovered")
				s.Assert().Len(suiteInfo[0].Tests, 1, "ValidSuite should have 1 test")
			},
		},
		{
			name: "Parse error returns error",
			files: map[string]string{
				"invalid.go": `package mypackage

this is not valid go code {{{`,
			},
			cfg:           &Config{},
			expectError:   true,
			errorContains: "failed to parse",
		},
		{
			name:           "Empty directory returns empty suite list",
			files:          map[string]string{},
			cfg:            &Config{},
			expectError:    false,
			expectedSuites: []string{},
		},
		{
			name: "Auto-detect mode without RunSuite returns error",
			files: map[string]string{
				"test.go": `package mypackage

import "github.com/stretchr/testify/suite"

type MySuite struct {
	suite.Suite
}

func (s *MySuite) TestA() {}`,
			},
			cfg: &Config{
				AutoDetectMode: true,
				GOFile:         "test.go",
			},
			expectError:   true,
			errorContains: "does not use depend.RunSuite",
		},
		{
			name: "Suite with dependencies",
			files: map[string]string{
				"test.go": `package mypackage

import "github.com/stretchr/testify/suite"
import "github.com/srgg/testify/depend"

type MySuite struct {
	suite.Suite
}

func (s *MySuite) TestA() {}

// @dependsOn TestA
func (s *MySuite) TestB() {}

// @dependsOn TestA, TestB
func (s *MySuite) TestC() {}

func TestRun(t *testing.T) {
	depend.RunSuite(t, new(MySuite))
}`,
			},
			cfg:            &Config{},
			expectError:    false,
			expectedSuites: []string{"MySuite"},
			validateTests: func(s *ScannerSuite, suiteInfo []SuiteInfo) {
				s.Assert().Len(suiteInfo, 1, "Should have 1 suite")
				tests := suiteInfo[0].Tests
				s.Assert().Len(tests, 3)
				s.Assert().Nil(tests[0].DependsOn, "TestA has no dependencies")
				s.Assert().Equal([]string{"TestA"}, tests[1].DependsOn, "TestB depends on TestA")
				s.Assert().Equal([]string{"TestA", "TestB"}, tests[2].DependsOn, "TestC depends on TestA and TestB")
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create test files
			fixture := NewTempDirFixture(s.T())
			var filePaths []string
			for filename, content := range tc.files {
				fixture.WithFile(filename, content)
				filePaths = append(filePaths, filepath.Join("", filename))
			}
			tmpDir := fixture.Build()

			// Build full file paths
			var fullPaths []string
			for filename := range tc.files {
				fullPaths = append(fullPaths, filepath.Join(tmpDir, filename))
			}

			tc.cfg.FilesToScan = fullPaths
			scanner := NewSuiteScanner(tc.cfg)

			suiteInfo, err := scanner.Scan()

			if tc.expectError {
				// All assertions MUST execute
				s.Assert().Error(err, "Expected error for: %s", tc.name)
				s.Assert().Contains(err.Error(), tc.errorContains, "Error message mismatch")
			} else {
				// All assertions MUST execute
				s.Assert().NoError(err, "Expected success for: %s", tc.name)

				// Extract suite names and create sorted comparison slice
				actualSuites := make([]string, 0, len(suiteInfo))
				for _, info := range suiteInfo {
					actualSuites = append(actualSuites, info.Name)
				}
				sort.Strings(actualSuites)
				sort.Strings(tc.expectedSuites)
				s.Assert().Equal(tc.expectedSuites, actualSuites, "Suite names mismatch")

				// Run custom test validation
				if tc.validateTests != nil {
					tc.validateTests(s, suiteInfo)
				}
			}
		})
	}
}

func TestScannerSuite(t *testing.T) {
	suite.Run(t, new(ScannerSuite))
}
