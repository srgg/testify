package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SelectionSuite struct {
	suite.Suite
}

func (s *SelectionSuite) TestSelectAutoDetectSuite() {
	// GOAL: Verify selectAutoDetect handles all auto-detect mode scenarios
	//
	// TEST SCENARIO: Various suite lists + auto-detect mode → selectAutoDetect called → Correct selection or error

	type testCase struct {
		name           string
		gofile         string
		suiteList      []SuiteInfo
		expectError    bool
		errorContains  string
		expectedSuites []string
	}

	testCases := []testCase{
		{
			name:           "Single suite returns that suite",
			gofile:         "test.go",
			suiteList:      []SuiteInfo{{Name: "MySuite", Package: "pkg"}},
			expectError:    false,
			expectedSuites: []string{"MySuite"},
		},
		{
			name:          "Empty suite list returns error",
			gofile:        "test.go",
			suiteList:     []SuiteInfo{},
			expectError:   true,
			errorContains: "no test suite found in test.go",
		},
		{
			name:   "Multiple suites returns error with suggestion",
			gofile: "test.go",
			suiteList: []SuiteInfo{
				{Name: "FirstSuite", Package: "pkg"},
				{Name: "SecondSuite", Package: "pkg"},
			},
			expectError:   true,
			errorContains: "multiple test suites found in test.go: [FirstSuite SecondSuite]",
		},
		{
			name:   "Multiple suites error includes first suite as example",
			gofile: "complex_test.go",
			suiteList: []SuiteInfo{
				{Name: "AlphaSuite", Package: "pkg"},
				{Name: "BetaSuite", Package: "pkg"},
				{Name: "GammaSuite", Package: "pkg"},
			},
			expectError:   true,
			errorContains: "//go:generate go run github.com/srgg/testify/depend/cmd/dependgen AlphaSuite",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cfg := &Config{GOFile: tc.gofile}
			selector := NewSuiteSelector(cfg)
			result, err := selector.selectAutoDetect(tc.suiteList)

			if tc.expectError {
				// All assertions MUST execute
				s.Assert().Error(err, "Expected error for: %s", tc.name)
				s.Assert().Contains(err.Error(), tc.errorContains, "Error message mismatch")
				s.Assert().Nil(result, "Result should be nil on error")
			} else {
				// All assertions MUST execute
				s.Assert().NoError(err, "Expected success for: %s", tc.name)
				// Extract suite names for comparison
				var resultNames []string
				for _, info := range result {
					resultNames = append(resultNames, info.Name)
				}
				s.Assert().Equal(tc.expectedSuites, resultNames, "Suite selection mismatch")
			}
		})
	}
}

func (s *SelectionSuite) TestSelectSuitesToGenerate() {
	// GOAL: Verify Select handles all operation modes correctly
	//
	// TEST SCENARIO: Different configs + suite slices → Select called → Correct suites selected

	type testCase struct {
		name           string
		cfg            *Config
		suiteInfo      []SuiteInfo
		expectError    bool
		errorContains  string
		expectedSuites []string
	}

	testCases := []testCase{
		{
			name: "Explicit target suite returns that suite",
			cfg: &Config{
				TargetSuite: "MySuite",
			},
			suiteInfo: []SuiteInfo{
				{Name: "MySuite", Package: "pkg"},
				{Name: "OtherSuite", Package: "pkg"},
			},
			expectError:    false,
			expectedSuites: []string{"MySuite"},
		},
		{
			name: "Auto-detect mode with single suite returns that suite",
			cfg: &Config{
				AutoDetectMode: true,
				GOFile:         "test.go",
			},
			suiteInfo: []SuiteInfo{
				{Name: "MySuite", Package: "pkg"},
			},
			expectError:    false,
			expectedSuites: []string{"MySuite"},
		},
		{
			name: "Auto-detect mode with no suites returns error",
			cfg: &Config{
				AutoDetectMode: true,
				GOFile:         "test.go",
			},
			suiteInfo:     []SuiteInfo{},
			expectError:   true,
			errorContains: "no test suite found in test.go",
		},
		{
			name: "Auto-detect mode with multiple suites returns error",
			cfg: &Config{
				AutoDetectMode: true,
				GOFile:         "test.go",
			},
			suiteInfo: []SuiteInfo{
				{Name: "FirstSuite", Package: "pkg"},
				{Name: "SecondSuite", Package: "pkg"},
			},
			expectError:   true,
			errorContains: "multiple test suites found",
		},
		{
			name: "Multi-suite mode returns all suites sorted",
			cfg: &Config{
				AutoDetectMode: false,
				TargetSuite:    "",
			},
			suiteInfo: []SuiteInfo{
				{Name: "AlphaSuite", Package: "pkg"},
				{Name: "BetaSuite", Package: "pkg"},
				{Name: "ZetaSuite", Package: "pkg"},
			},
			expectError:    false,
			expectedSuites: []string{"AlphaSuite", "BetaSuite", "ZetaSuite"},
		},
		{
			name: "Multi-suite mode with no suites returns error",
			cfg: &Config{
				AutoDetectMode: false,
				TargetSuite:    "",
			},
			suiteInfo:     []SuiteInfo{},
			expectError:   true,
			errorContains: "no test suites found that use depend.RunSuite",
		},
		{
			name: "Multi-suite mode with single suite returns that suite",
			cfg: &Config{
				AutoDetectMode: false,
				TargetSuite:    "",
			},
			suiteInfo: []SuiteInfo{
				{Name: "OnlySuite", Package: "pkg"},
			},
			expectError:    false,
			expectedSuites: []string{"OnlySuite"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			selector := NewSuiteSelector(tc.cfg)
			result, err := selector.Select(tc.suiteInfo)

			if tc.expectError {
				// All assertions MUST execute
				s.Assert().Error(err, "Expected error for: %s", tc.name)
				s.Assert().Contains(err.Error(), tc.errorContains, "Error message mismatch")
				s.Assert().Nil(result, "Result should be nil on error")
			} else {
				// All assertions MUST execute
				s.Assert().NoError(err, "Expected success for: %s", tc.name)
				// Extract suite names for comparison
				var resultNames []string
				for _, info := range result {
					resultNames = append(resultNames, info.Name)
				}
				s.Assert().Equal(tc.expectedSuites, resultNames, "Suite selection mismatch")
			}
		})
	}
}

func TestSelectionSuite(t *testing.T) {
	suite.Run(t, new(SelectionSuite))
}
