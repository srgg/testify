package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) TestParseConfig() {
	// GOAL: Verify parseConfig correctly handles all operation modes and edge cases
	//
	// TEST SCENARIO: Parse flags and environment → Config populated → Mode correctly determined

	type testCase struct {
		name              string
		args              []string
		gofile            string
		files             []string
		expectAutoDetect  bool
		expectTargetSuite string
		expectGOFile      string
		expectFilesCount  int
		expectVerbose     bool
		customValidation  func(*ConfigSuite, *Config, string)
	}

	scenarioGroups := []struct {
		group string
		cases []testCase
	}{
		{
			group: "Operation Modes",
			cases: []testCase{
				{
					name:              "AutoDetect mode with GOFILE",
					args:              []string{"dependgen"},
					gofile:            "test_test.go",
					files:             []string{"test_test.go"},
					expectAutoDetect:  true,
					expectTargetSuite: "",
					expectGOFile:      "test_test.go",
					expectFilesCount:  1,
					expectVerbose:     false,
				},
				{
					name:              "Explicit suite mode",
					args:              []string{"dependgen", "MySuite"},
					gofile:            "",
					files:             []string{"test1_test.go", "test2_test.go"},
					expectAutoDetect:  false,
					expectTargetSuite: "MySuite",
					expectGOFile:      "",
					expectFilesCount:  2,
					expectVerbose:     false,
				},
				{
					name:              "Multi-suite mode",
					args:              []string{"dependgen"},
					gofile:            "",
					files:             []string{"test1_test.go", "test2_test.go"},
					expectAutoDetect:  false,
					expectTargetSuite: "",
					expectGOFile:      "",
					expectFilesCount:  2,
					expectVerbose:     false,
				},
			},
		},
		{
			group: "Flags",
			cases: []testCase{
				{
					name:              "Verbose flag",
					args:              []string{"dependgen", "-v"},
					gofile:            "",
					files:             []string{},
					expectAutoDetect:  false,
					expectTargetSuite: "",
					expectGOFile:      "",
					expectFilesCount:  0,
					expectVerbose:     true,
				},
			},
		},
		{
			group: "Edge Cases",
			cases: []testCase{
				{
					name:              "No test files in directory",
					args:              []string{"dependgen"},
					gofile:            "",
					files:             []string{},
					expectAutoDetect:  false,
					expectTargetSuite: "",
					expectGOFile:      "",
					expectFilesCount:  0,
					expectVerbose:     false,
				},
				{
					name:              "Multiple arguments provided",
					args:              []string{"dependgen", "FirstSuite", "SecondSuite", "ThirdSuite"},
					gofile:            "",
					files:             []string{},
					expectAutoDetect:  false,
					expectTargetSuite: "FirstSuite",
					expectGOFile:      "",
					expectFilesCount:  0,
					expectVerbose:     false,
				},
				{
					name:              "GOFILE and suite name both provided",
					args:              []string{"dependgen", "ExplicitSuite"},
					gofile:            "test1_test.go",
					files:             []string{"test1_test.go", "test2_test.go"},
					expectAutoDetect:  false,
					expectTargetSuite: "ExplicitSuite",
					expectGOFile:      "test1_test.go",
					expectFilesCount:  2,
					expectVerbose:     false,
				},
				{
					name:              "Invalid GOFILE path",
					args:              []string{"dependgen"},
					gofile:            "nonexistent_test.go",
					files:             []string{},
					expectAutoDetect:  true,
					expectTargetSuite: "",
					expectGOFile:      "nonexistent_test.go",
					expectFilesCount:  1,
					expectVerbose:     false,
					customValidation: func(s *ConfigSuite, cfg *Config, testDir string) {
						// testDir is already resolved by TempDirFixture
						expectedPath := filepath.Join(testDir, "nonexistent_test.go")
						s.Assert().Equal(expectedPath, cfg.FilesToScan[0], "MUST create path even if file doesn't exist")

						// Verify the file doesn't exist
						_, err := os.Stat(cfg.FilesToScan[0])
						s.Assert().Error(err, "MUST confirm file doesn't exist")
						s.Assert().True(os.IsNotExist(err), "MUST be file not found error")
					},
				},
			},
		},
	}

	for _, group := range scenarioGroups {
		s.Run(group.group, func() {
			for _, tc := range group.cases {
				s.Run(tc.name, func() {
					// Reset flags and set up the environment
					flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
					os.Args = tc.args
					NewEnvFixture(s.T()).Set("GOFILE", tc.gofile)

					// Create a test directory with files and change to it
					builder := NewTempDirFixture(s.T()).WithChangeDir()
					for _, file := range tc.files {
						builder.WithFile(file, "package test")
					}
					testDir := builder.Build()

					// Parse config
					cfg, err := parseConfig()

					// All assertions MUST execute
					s.Assert().NoError(err, "MUST parse config without error")
					s.Assert().Equal(tc.expectAutoDetect, cfg.AutoDetectMode, "AutoDetectMode mismatch")
					s.Assert().Equal(tc.expectTargetSuite, cfg.TargetSuite, "TargetSuite mismatch")
					s.Assert().Equal(tc.expectGOFile, cfg.GOFile, "GOFile mismatch")
					s.Assert().Equal(testDir, cfg.WorkingDir, "WorkingDir mismatch")
					s.Assert().Len(cfg.FilesToScan, tc.expectFilesCount, "FilesToScan count mismatch")
					s.Assert().Equal(tc.expectVerbose, cfg.Verbose, "Verbose mismatch")

					// Run custom validation if provided
					if tc.customValidation != nil {
						tc.customValidation(s, cfg, testDir)
					}
				})
			}
		})
	}
}

func (s *ConfigSuite) TestParseConfigGOFILESecurityValidation() {
	// GOAL: Verify GOFILE security validation prevents path traversal and injection attacks
	//
	// TEST SCENARIO: Set malicious GOFILE values → parseConfig called → Returns security error

	type testCase struct {
		name          string
		gofile        string
		expectedError string
	}

	testCases := []testCase{
		{
			name:          "Absolute Unix path rejected",
			gofile:        "/etc/passwd",
			expectedError: "invalid GOFILE: must be a relative filename, got absolute path: /etc/passwd",
		},
		{
			name:          "Parent directory traversal rejected",
			gofile:        "../../../etc/passwd",
			expectedError: "invalid GOFILE: path traversal not allowed: ../../../etc/passwd",
		},
		{
			name:          "Single parent directory traversal rejected",
			gofile:        "../test.go",
			expectedError: "invalid GOFILE: path traversal not allowed: ../test.go",
		},
		{
			name:          "Directory component rejected",
			gofile:        "subdir/test.go",
			expectedError: "invalid GOFILE: must be a filename in current directory, got: subdir/test.go",
		},
		{
			name:          "Hidden directory traversal rejected",
			gofile:        "test..go",
			expectedError: "invalid GOFILE: path traversal not allowed: test..go",
		},
		{
			name:          "Nested directory traversal rejected",
			gofile:        "a/../../../test.go",
			expectedError: "invalid GOFILE: path traversal not allowed: a/../../../test.go",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = []string{"dependgen"}
			NewEnvFixture(s.T()).Set("GOFILE", tc.gofile)

			// Create test directory and change to it
			NewTempDirFixture(s.T()).WithChangeDir().Build()

			// Parse config - should fail with security error
			cfg, err := parseConfig()

			// All assertions MUST execute
			s.Assert().Error(err, "MUST reject malicious GOFILE: %s", tc.gofile)
			s.Assert().Contains(err.Error(), tc.expectedError, "Error message mismatch")
			s.Assert().Nil(cfg, "MUST return nil config on security error")
		})
	}
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
