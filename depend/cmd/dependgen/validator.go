package main

import (
	"fmt"
	"strings"
)

// DependencyValidator handles validation of test method dependencies.
// It checks for invalid dependencies, circular dependencies, and other dependency-related issues.
type DependencyValidator struct {
	cfg *Config
}

// NewDependencyValidator creates a new dependency validator.
func NewDependencyValidator(cfg *Config) *DependencyValidator {
	return &DependencyValidator{cfg: cfg}
}

// Validate validates dependencies for all suites to be generated.
// Returns an error if validation fails for any suite.
func (v *DependencyValidator) Validate(suitesToGenerate []SuiteInfo) error {
	verbosef(v.cfg, fmt.Sprintf("Validating dependencies for %d suite(s)...", len(suitesToGenerate)))

	for _, suite := range suitesToGenerate {
		if len(suite.Tests) == 0 {
			continue
		}

		if err := v.validateSuite(suite.Name, suite.Tests); err != nil {
			return fmt.Errorf("dependency validation failed for suite %s:\n\n%w", suite.Name, err)
		}
	}
	return nil
}

// validateSuite validates dependencies for a single suite.
func (v *DependencyValidator) validateSuite(_ string, tests []TestMethodInfo) error {
	testNames := v.buildTestNameSet(tests)
	var errors []string

	for _, test := range tests {
		for _, dep := range test.DependsOn {
			if dep == test.Name {
				errors = append(errors, fmt.Sprintf("Invalid dependency in %s: test cannot depend on itself", test.Name))
				continue
			}
			if _, exists := testNames[dep]; !exists {
				errors = append(errors, fmt.Sprintf("Invalid dependency in %s: %s does not exist", test.Name, dep))
			}
		}
	}

	if circularPath := v.detectCircularDependencies(tests); circularPath != nil {
		errors = append(errors, fmt.Sprintf("Circular dependency detected: %s", v.formatCircularPath(circularPath)))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n"))
	}
	return nil
}

// buildTestNameSet creates a set of test names for quick lookup.
func (v *DependencyValidator) buildTestNameSet(tests []TestMethodInfo) map[string]struct{} {
	testNames := make(map[string]struct{})
	for _, test := range tests {
		testNames[test.Name] = struct{}{}
	}
	return testNames
}

// detectCircularDependencies uses DFS to detect cycles in the dependency graph.
// Returns the cycle path if found, nil otherwise.
//
// The cycle path shows the complete cycle, e.g., ["TestA", "TestB", "TestC", "TestA"]
// indicates TestA → TestB → TestC → TestA.
func (v *DependencyValidator) detectCircularDependencies(tests []TestMethodInfo) []string {
	// Build adjacency list
	graph := make(map[string][]string)

	for _, test := range tests {
		graph[test.Name] = test.DependsOn
	}

	// Track visited nodes and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	// DFS helper function
	var dfs func(node string) []string
	dfs = func(node string) []string {
		visited[node] = true
		recStack[node] = true

		for _, dep := range graph[node] {
			if !visited[dep] {
				parent[dep] = node
				if cycle := dfs(dep); cycle != nil {
					return cycle
				}
			} else if recStack[dep] {
				// Found a cycle - build the path from dep to node by tracing back
				var path []string
				current := node
				for current != dep {
					path = append([]string{current}, path...) // Prepend
					current = parent[current]
				}
				// Now path goes from node back to dep, but we want dep → ... → node → dep
				cycle := append([]string{dep}, path...)
				cycle = append(cycle, dep) // Complete the cycle
				return cycle
			}
		}

		recStack[node] = false
		return nil
	}

	// Check each node
	for _, test := range tests {
		if !visited[test.Name] {
			if cycle := dfs(test.Name); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}

// formatCircularPath formats a circular dependency path for error messages.
// Example: ["TestA", "TestB", "TestA"] → "TestA → TestB → TestA"
func (v *DependencyValidator) formatCircularPath(path []string) string {
	return strings.Join(path, " → ")
}
