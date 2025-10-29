// Package documentation is in doc.go
package main

import (
	"fmt"
	"log"
	"os"
)

// main executes the code generation workflow.
// It parses configuration, scans files, validates dependencies, and generates code.
func main() {
	// Parse configuration
	cfg, err := parseConfig()
	if err != nil {
		fatalf("%v", err)
	}

	verbosef(cfg, fmt.Sprintf(`Configuration:
  Working directory: %s
  Target suite: %s
  Auto-detect mode: %v
  Scanning %d file(s)...`, cfg.WorkingDir, cfg.TargetSuite, cfg.AutoDetectMode, len(cfg.FilesToScan)))

	// Scan files and discover suites
	scanner := NewSuiteScanner(cfg)
	suiteInfo, err := scanner.Scan()
	if err != nil {
		fatalf("%v", err)
	}

	// Determine which suites to generate
	selector := NewSuiteSelector(cfg)
	suitesToGenerate, err := selector.Select(suiteInfo)
	if err != nil {
		fatalf("%v", err)
	}

	// Validate dependencies
	validator := NewDependencyValidator(cfg)
	err = validator.Validate(suitesToGenerate)
	if err != nil {
		fatalf("%v", err)
	}

	// Create generator and generate code files
	generator := NewGenerator(cfg)
	err = generator.Generate(suitesToGenerate)
	if err != nil {
		fatalf("%v", err)
	}
}

// verbosef prints messages if verbose mode is enabled.
// It accepts multiple format strings and their arguments.
func verbosef(cfg *Config, messages ...string) {
	if cfg.Verbose {
		for _, msg := range messages {
			log.Println(msg)
		}
	}
}

// fatalf prints an error message to stderr and exits with code 1.
// It follows the pattern used by standard Go tools (gofmt, go vet, etc).
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dependgen: "+format+"\n", args...)
	os.Exit(1)
}
