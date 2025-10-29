package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ValidatorSuite struct {
	suite.Suite
}

func (s *ValidatorSuite) TestDetectCircularDependencies() {
	testCases := []struct {
		name          string
		tests         []TestMethodInfo
		expectedCycle []string
	}{
		{
			name: "Linear dependency chain returns nil",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: nil},
				{Name: "TestB", DependsOn: []string{"TestA"}},
				{Name: "TestC", DependsOn: []string{"TestB"}},
			},
			expectedCycle: nil,
		},
		{
			name: "Direct circular dependency detects cycle",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: []string{"TestB"}},
				{Name: "TestB", DependsOn: []string{"TestA"}},
			},
			expectedCycle: []string{"TestA", "TestB", "TestA"},
		},
		{
			name: "Transitive circular dependency detects cycle",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: []string{"TestB"}},
				{Name: "TestB", DependsOn: []string{"TestC"}},
				{Name: "TestC", DependsOn: []string{"TestA"}},
			},
			expectedCycle: []string{"TestA", "TestB", "TestC", "TestA"},
		},
		{
			name: "Self dependency detects cycle",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: []string{"TestA"}},
			},
			expectedCycle: []string{"TestA", "TestA"},
		},
		{
			name: "Complex graph with multiple dependencies returns nil",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: nil},
				{Name: "TestB", DependsOn: []string{"TestA"}},
				{Name: "TestC", DependsOn: []string{"TestB"}},
				{Name: "TestD", DependsOn: []string{"TestC"}},
				{Name: "TestE", DependsOn: []string{"TestD", "TestB"}}, // Multiple deps but no cycle
			},
			expectedCycle: nil,
		},
		{
			name: "Four node cycle detects full path",
			tests: []TestMethodInfo{
				{Name: "TestA", DependsOn: []string{"TestB"}},
				{Name: "TestB", DependsOn: []string{"TestC"}},
				{Name: "TestC", DependsOn: []string{"TestD"}},
				{Name: "TestD", DependsOn: []string{"TestA"}}, // Creates cycle: A→B→C→D→A
			},
			expectedCycle: []string{"TestA", "TestB", "TestC", "TestD", "TestA"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			validator := NewDependencyValidator(&Config{})
			cycle := validator.detectCircularDependencies(tc.tests)
			if tc.expectedCycle == nil {
				s.Assert().Nil(cycle, "Expected no cycle")
			} else {
				s.Assert().NotNil(cycle, "Expected a cycle to be detected")
				s.Assert().Equal(tc.expectedCycle, cycle, "Cycle path should match")
			}
		})
	}
}

func (s *ValidatorSuite) TestFormatCircularPath() {
	testCases := []struct {
		name     string
		path     []string
		expected string
	}{
		{
			name:     "Direct cycle",
			path:     []string{"TestA", "TestB", "TestA"},
			expected: "TestA → TestB → TestA",
		},
		{
			name:     "Transitive cycle",
			path:     []string{"TestA", "TestB", "TestC", "TestA"},
			expected: "TestA → TestB → TestC → TestA",
		},
		{
			name:     "Self cycle",
			path:     []string{"TestA", "TestA"},
			expected: "TestA → TestA",
		},
		{
			name:     "Long cycle",
			path:     []string{"TestA", "TestB", "TestC", "TestD", "TestE", "TestA"},
			expected: "TestA → TestB → TestC → TestD → TestE → TestA",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			validator := NewDependencyValidator(&Config{})
			result := validator.formatCircularPath(tc.path)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *ValidatorSuite) TestValidate() {
	// GOAL: Verify Validate() orchestrates validation across multiple suites correctly
	//
	// TEST SCENARIO: Various suite configurations → Validate() called → Correct error handling and reporting

	type testCase struct {
		name             string
		suitesToValidate []SuiteInfo
		expectError      bool
		errorContains    []string // Multiple strings that should appear in error
	}

	testCases := []testCase{
		{
			name: "All suites valid returns no error",
			suitesToValidate: []SuiteInfo{
				{
					Name: "FirstSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: nil},
						{Name: "TestB", DependsOn: []string{"TestA"}},
					},
				},
				{
					Name: "SecondSuite",
					Tests: []TestMethodInfo{
						{Name: "TestX", DependsOn: nil},
						{Name: "TestY", DependsOn: []string{"TestX"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "First suite invalid stops validation",
			suitesToValidate: []SuiteInfo{
				{
					Name: "InvalidSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: []string{"TestB"}},
						{Name: "TestB", DependsOn: []string{"NonExistent"}},
					},
				},
				{
					Name: "ValidSuite",
					Tests: []TestMethodInfo{
						{Name: "TestX", DependsOn: nil},
					},
				},
			},
			expectError:   true,
			errorContains: []string{"dependency validation failed for suite InvalidSuite", "NonExistent does not exist"},
		},
		{
			name: "Second suite invalid detected",
			suitesToValidate: []SuiteInfo{
				{
					Name: "ValidSuite",
					Tests: []TestMethodInfo{
						{Name: "TestX", DependsOn: nil},
					},
				},
				{
					Name: "InvalidSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: []string{"TestA"}}, // Self-dependency
					},
				},
			},
			expectError:   true,
			errorContains: []string{"dependency validation failed for suite InvalidSuite", "test cannot depend on itself"},
		},
		{
			name: "Multiple error types in single suite combined",
			suitesToValidate: []SuiteInfo{
				{
					Name: "ComplexSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: []string{"TestA"}},       // Self-dependency
						{Name: "TestB", DependsOn: []string{"NonExistent"}}, // Non-existent
						{Name: "TestC", DependsOn: []string{"TestD"}},       // Will create circular
						{Name: "TestD", DependsOn: []string{"TestC"}},       // Circular with TestC
						{Name: "TestE", DependsOn: []string{"AlsoMissing"}}, // Another non-existent
					},
				},
			},
			expectError: true,
			errorContains: []string{
				"dependency validation failed for suite ComplexSuite",
				"test cannot depend on itself",
				"NonExistent does not exist",
				"Circular dependency detected",
				"TestA → TestA", // Self-dep creates a cycle
			},
		},
		{
			name: "Empty suite skipped during validation",
			suitesToValidate: []SuiteInfo{
				{
					Name:  "EmptySuite",
					Tests: []TestMethodInfo{},
				},
				{
					Name: "ValidSuite",
					Tests: []TestMethodInfo{
						{Name: "TestX", DependsOn: nil},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Circular dependency error includes formatted path",
			suitesToValidate: []SuiteInfo{
				{
					Name: "CircularSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: []string{"TestB"}},
						{Name: "TestB", DependsOn: []string{"TestC"}},
						{Name: "TestC", DependsOn: []string{"TestA"}},
					},
				},
			},
			expectError:   true,
			errorContains: []string{"Circular dependency detected", "TestA → TestB → TestC → TestA"},
		},
		{
			name: "Single suite with only self-dependency error",
			suitesToValidate: []SuiteInfo{
				{
					Name: "SelfDepSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: nil},
						{Name: "TestB", DependsOn: []string{"TestB"}}, // Self-dep
						{Name: "TestC", DependsOn: []string{"TestA"}},
					},
				},
			},
			expectError:   true,
			errorContains: []string{"dependency validation failed for suite SelfDepSuite", "TestB", "test cannot depend on itself"},
		},
		{
			name: "Single suite with only non-existent dependency error",
			suitesToValidate: []SuiteInfo{
				{
					Name: "NonExistSuite",
					Tests: []TestMethodInfo{
						{Name: "TestA", DependsOn: []string{"MissingTest"}},
					},
				},
			},
			expectError:   true,
			errorContains: []string{"dependency validation failed for suite NonExistSuite", "MissingTest does not exist"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			validator := NewDependencyValidator(&Config{})
			err := validator.Validate(tc.suitesToValidate)

			if tc.expectError {
				// All assertions MUST execute
				s.Assert().Error(err, "Expected error for: %s", tc.name)
				for _, expectedMsg := range tc.errorContains {
					s.Assert().Contains(err.Error(), expectedMsg, "Error should contain: %s", expectedMsg)
				}
			} else {
				// All assertions MUST execute
				s.Assert().NoError(err, "Expected success for: %s", tc.name)
			}
		})
	}
}

func TestValidatorSuite(t *testing.T) {
	suite.Run(t, new(ValidatorSuite))
}
