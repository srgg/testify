package main

import (
	"fmt"
)

// SuiteSelector handles suite selection based on configuration mode.
// It determines which test suites to generate code for based on explicit targets,
// auto-detection, or multi-suite mode.
type SuiteSelector struct {
	cfg *Config
}

// NewSuiteSelector creates a new suite selector with the given configuration.
func NewSuiteSelector(cfg *Config) *SuiteSelector {
	return &SuiteSelector{cfg: cfg}
}

// Select determines which suites to generate based on the configuration mode.
// Returns the selected suite information and any error encountered.
func (s *SuiteSelector) Select(suiteInfo []SuiteInfo) ([]SuiteInfo, error) {
	if s.cfg.TargetSuite != "" {
		// Linear search for the target suite
		for _, info := range suiteInfo {
			if info.Name == s.cfg.TargetSuite {
				return []SuiteInfo{info}, nil
			}
		}
		return nil, fmt.Errorf("suite %s not found", s.cfg.TargetSuite)
	}

	if s.cfg.AutoDetectMode {
		return s.selectAutoDetect(suiteInfo)
	}

	if len(suiteInfo) == 0 {
		return nil, fmt.Errorf("no test suites found that use depend.RunSuite")
	}
	return suiteInfo, nil
}

// selectAutoDetect handles suite selection in auto-detect mode.
// Returns the selected suite or an error if selection fails.
func (s *SuiteSelector) selectAutoDetect(suiteList []SuiteInfo) ([]SuiteInfo, error) {
	if len(suiteList) == 0 {
		return nil, fmt.Errorf("no test suite found in %s\n\n"+
			"A test suite must be a struct that embeds suite.Suite", s.cfg.GOFile)
	}
	if len(suiteList) > 1 {
		names := make([]string, len(suiteList))
		for i, info := range suiteList {
			names[i] = info.Name
		}
		return nil, fmt.Errorf("multiple test suites found in %s: %v\n\n"+
			"Specify which suite to generate:\n"+
			"  //go:generate go run github.com/srgg/testify/depend/cmd/dependgen %s",
			s.cfg.GOFile, names, names[0])
	}
	return suiteList, nil
}
