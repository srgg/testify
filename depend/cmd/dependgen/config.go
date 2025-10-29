package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the configuration for the dependgen tool.
type Config struct {
	TargetSuite    string   // Specific suite to generate (empty for auto-detect or multi-suite)
	WorkingDir     string   // Current working directory
	AutoDetectMode bool     // Whether running in go:generate auto-detect mode
	GOFile         string   // GOFILE environment variable value
	FilesToScan    []string // List of test files to scan
	Verbose        bool     // Enable verbose logging
}

// parseConfig parses command-line arguments and environment variables
// to build the code generation configuration.
func parseConfig() (*Config, error) {
	cfg := &Config{}

	// Parse flags
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	cfg.Verbose = *verbose

	// Parse command-line arguments (non-flag arguments)
	args := flag.Args()
	if len(args) > 0 {
		cfg.TargetSuite = args[0]
	}

	// Get working directory
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	cfg.WorkingDir = dir

	// Check for the GOFILE environment variable (set by go:generate)
	cfg.GOFile = os.Getenv("GOFILE")

	// Validate GOFILE for security (defense in depth against path traversal)
	if cfg.GOFile != "" {
		if filepath.IsAbs(cfg.GOFile) {
			return nil, fmt.Errorf("invalid GOFILE: must be a relative filename, got absolute path: %s", cfg.GOFile)
		}
		if strings.Contains(cfg.GOFile, "..") {
			return nil, fmt.Errorf("invalid GOFILE: path traversal not allowed: %s", cfg.GOFile)
		}
		// Normalize to ensure it's just a filename without directory components
		if filepath.Dir(cfg.GOFile) != "." {
			return nil, fmt.Errorf("invalid GOFILE: must be a filename in current directory, got: %s", cfg.GOFile)
		}
	}

	cfg.AutoDetectMode = cfg.TargetSuite == "" && cfg.GOFile != ""

	// Determine which files to scan
	if cfg.TargetSuite == "" {
		if cfg.GOFile != "" {
			// GOFILE set by go:generate - scan only that file
			cfg.FilesToScan = []string{filepath.Join(dir, cfg.GOFile)}
		} else {
			// No GOFILE - scan all test files
			files, err := scanTestFiles(dir)
			if err != nil {
				return nil, err
			}
			cfg.FilesToScan = files
		}
	} else {
		// Explicit suite mode: scan all test files
		files, err := scanTestFiles(dir)
		if err != nil {
			return nil, err
		}
		cfg.FilesToScan = files
	}

	return cfg, nil
}

// scanTestFiles scans for all *_test.go files in the given directory.
// Returns a list of absolute paths to test files found.
func scanTestFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan test files: %w", err)
	}
	return files, nil
}
