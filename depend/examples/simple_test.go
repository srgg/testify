// Package examples demonstrates testify/depend usage for test dependency management.
//
// This example shows:
//   - Declaring test dependencies with @dependsOn comments
//   - Auto-generating dependency configuration with dependgen
//   - Running test suites with depend.RunSuite
//   - Automatic test skipping when dependencies fail
//
// Generate dependency code:
//
//	go generate ./...
//
// Run the example:
//
//	go test -v
//
// Expected output demonstrates dependency behavior:
//
// --- FAIL: TestSimpleSuite (0.00s)
//
//	--- PASS: TestSimpleSuite/All_pass_-_no_failures (0.00s)
//	    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestA (0.00s)
//	    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestB (0.00s)
//	    --- PASS: TestSimpleSuite/All_pass_-_no_failures/TestC (0.00s)
//	--- FAIL: TestSimpleSuite/A_fails_-_B_and_C_skipped (0.00s)
//	    --- FAIL: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestA (0.00s)
//	    --- SKIP: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestB (0.00s)
//	    --- SKIP: TestSimpleSuite/A_fails_-_B_and_C_skipped/TestC (0.00s)
//	--- FAIL: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped (0.00s)
//	    --- PASS: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestA (0.00s)
//	    --- FAIL: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestB (0.00s)
//	    --- SKIP: TestSimpleSuite/A_passes,_B_fails_-_only_C_skipped/TestC (0.00s)

package examples

import (
	"testing"

	"github.com/srgg/testify/depend"
	"github.com/stretchr/testify/suite"
)

//go:generate go run github.com/srgg/testify/depend/cmd/dependgen

type SimpleSuite struct {
	suite.Suite

	shouldFailA bool
	shouldFailB bool
	dataA       string
	dataB       string
}

func (s *SimpleSuite) TestA() {
	if s.shouldFailA {
		s.Fail("TestA intentionally failed")
		return
	}

	s.dataA = "data-from-A"
	s.Assert().Equal("data-from-A", s.dataA)
}

// @dependsOn TestA
func (s *SimpleSuite) TestB() {
	if s.shouldFailB {
		s.Fail("TestB intentionally failed")
		return
	}

	// This test depends on TestA
	// It uses data produced by TestA
	s.dataB = s.dataA + "-processed-by-B"
	s.Assert().Equal("data-from-A-processed-by-B", s.dataB)
}

// @dependsOn TestA, TestB
func (s *SimpleSuite) TestC() {
	// This test depends on BOTH TestA and TestB
	// It uses data from both tests
	dataC := s.dataA + "-and-" + s.dataB
	s.Assert().Equal("data-from-A-and-data-from-A-processed-by-B", dataC)
}

func TestSimpleSuite(t *testing.T) {
	testCases := []struct {
		name        string
		shouldFailA bool
		shouldFailB bool
		expectA     bool // true = pass, false = fail
		expectB     bool // true = pass, false = fail/skip
		expectC     bool // true = pass, false = fail/skip
	}{
		{
			name:        "All pass - no failures",
			shouldFailA: false,
			shouldFailB: false,
			expectA:     true,
			expectB:     true,
			expectC:     true,
		},
		{
			name:        "A fails - B and C skipped",
			shouldFailA: true,
			shouldFailB: false,
			expectA:     false,
			expectB:     false, // skipped because A failed
			expectC:     false, // skipped because A failed
		},
		{
			name:        "A passes, B fails - only C skipped",
			shouldFailA: false,
			shouldFailB: true,
			expectA:     true,
			expectB:     false,
			expectC:     false, // skipped because B failed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &SimpleSuite{
				shouldFailA: tc.shouldFailA,
				shouldFailB: tc.shouldFailB,
			}

			result := depend.RunSuite(t, s)

			// Verify TestA result
			expectAStatus := depend.TestPassed
			if !tc.expectA {
				expectAStatus = depend.TestFailed
			}
			if result.Tests["TestA"].Status != expectAStatus {
				t.Errorf("TestA: expected %v, got %v", expectAStatus, result.Tests["TestA"].Status)
			}

			// Verify TestB result
			if tc.expectB {
				if result.Tests["TestB"].Status != depend.TestPassed {
					t.Errorf("TestB: expected Pass, got %v", result.Tests["TestB"].Status)
				}
			} else {
				// Either Failed or Skipped is acceptable for false expectation
				if result.Tests["TestB"].Status != depend.TestFailed && result.Tests["TestB"].Status != depend.TestSkipped {
					t.Errorf("TestB: expected Failed or Skipped, got %v", result.Tests["TestB"].Status)
				}
			}

			// Verify TestC result
			if tc.expectC {
				if result.Tests["TestC"].Status != depend.TestPassed {
					t.Errorf("TestC: expected Pass, got %v", result.Tests["TestC"].Status)
				}
			} else {
				if result.Tests["TestC"].Status != depend.TestFailed && result.Tests["TestC"].Status != depend.TestSkipped {
					t.Errorf("TestC: expected Failed or Skipped, got %v", result.Tests["TestC"].Status)
				}
			}

			t.Logf("✓ Results: TestA=%v TestB=%v TestC=%v",
				result.Tests["TestA"].Status, result.Tests["TestB"].Status, result.Tests["TestC"].Status)
		})
	}
}
