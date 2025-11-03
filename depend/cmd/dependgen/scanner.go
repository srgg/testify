package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// TestMethodInfo represents information about a test method for code generation.
type TestMethodInfo struct {
	Name      string   // Test method name (e.g., "TestFoo")
	DependsOn []string // List of test names this test depends on
}

// SuiteInfo holds information about a discovered test suite.
type SuiteInfo struct {
	Name       string           // Suite name (e.g., "TestifyLifecycleSuite")
	Package    string           // Package name (e.g., "depend")
	Tests      []TestMethodInfo // Test methods in this suite
	SourceFile string           // Source file where suite was defined (e.g., "runtime_test.go")
	BuildTags  []string         // Build tags from source file (e.g., ["//go:build test", "// +build test"])
}

// suiteCollector collects suite information during scanning.
// It encapsulates the logic for tracking suites and their test methods.
type suiteCollector struct {
	suites           map[string]SuiteInfo
	srcFileBuildTags map[string][]string // sourceFile -> buildTags
}

// newSuiteCollector creates a new suite collector.
func newSuiteCollector() *suiteCollector {
	return &suiteCollector{
		suites:           make(map[string]SuiteInfo),
		srcFileBuildTags: make(map[string][]string),
	}
}

// addSuiteName registers a suite name.
func (c *suiteCollector) addSuiteName(name string) {
	if _, exists := c.suites[name]; !exists {
		c.suites[name] = SuiteInfo{
			Name: name,
		}
	}
}

// setBuildTags stores build tags for a source file.
func (c *suiteCollector) setBuildTags(sourceFile string, buildTags []string) {
	c.srcFileBuildTags[sourceFile] = buildTags
}

// addTest adds a test method to a suite.
// If the suite doesn't have metadata set, it sets them.
func (c *suiteCollector) addTest(suiteName, packageName, sourceFile string, test TestMethodInfo) {
	info := c.suites[suiteName]
	if info.Name == "" {
		info.Name = suiteName
	}
	if info.Package == "" {
		info.Package = packageName
	}
	if info.SourceFile == "" {
		info.SourceFile = sourceFile
	}
	// Set build tags from the map if not already set
	if info.BuildTags == nil {
		info.BuildTags = c.srcFileBuildTags[sourceFile]
	}
	info.Tests = append(info.Tests, test)
	c.suites[suiteName] = info
}

// suiteNames returns a map of all collected suite names.
func (c *suiteCollector) suiteNames() map[string]struct{} {
	names := make(map[string]struct{}, len(c.suites))
	for name := range c.suites {
		names[name] = struct{}{}
	}
	return names
}

// result returns the final collected suite information as a sorted slice.
func (c *suiteCollector) result() []SuiteInfo {
	suiteList := make([]SuiteInfo, 0, len(c.suites))
	for _, info := range c.suites {
		suiteList = append(suiteList, info)
	}
	// Sort by name for consistent ordering
	sort.Slice(suiteList, func(i, j int) bool {
		return suiteList[i].Name < suiteList[j].Name
	})
	return suiteList
}

// SuiteScanner handles scanning and discovering test suites from Go source files.
// It performs AST analysis to find suite types and their test methods.
type SuiteScanner struct {
	cfg       *Config
	fset      *token.FileSet
	collector *suiteCollector
}

// NewSuiteScanner creates a new suite scanner with the given configuration.
func NewSuiteScanner(cfg *Config) *SuiteScanner {
	return &SuiteScanner{
		cfg:       cfg,
		fset:      token.NewFileSet(),
		collector: newSuiteCollector(),
	}
}

// Scan parses test files to discover suites and their test methods.
// Returns suite information (with source file tracking), and any error encountered.
func (s *SuiteScanner) Scan() ([]SuiteInfo, error) {
	for _, file := range s.cfg.FilesToScan {
		f, err := parser.ParseFile(s.fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		// Extract just the filename (e.g., "/path/to/runtime_test.go" → "runtime_test.go")
		sourceFile := filepath.Base(file)

		ok, err := s.processFile(f, sourceFile)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
	}

	s.logDiscoveredSuites()
	return s.collector.result(), nil
}

// processFile processes a single AST file to find suites and test methods.
// Returns false if the file should be skipped entirely, or an error if processing fails.
func (s *SuiteScanner) processFile(f *ast.File, sourceFile string) (bool, error) {
	fileUsesRunSuite := s.usesRunSuite(f)

	if s.cfg.AutoDetectMode && !fileUsesRunSuite {
		return false, fmt.Errorf("file %s does not use depend.RunSuite\n\n"+
			"Remove the //go:generate directive or add depend.RunSuite usage", s.cfg.GOFile)
	}

	if !fileUsesRunSuite {
		return false, nil
	}

	// Extract package name and build tags from the AST file
	packageName := f.Name.Name
	buildTags := extractBuildTags(f)

	// Store build tags for this source file
	s.collector.setBuildTags(sourceFile, buildTags)

	// Single-pass AST traversal: process both type declarations and method declarations in one iteration
	for _, decl := range f.Decls {
		s.processDeclaration(decl, packageName, sourceFile)
	}

	return true, nil
}

// processDeclaration handles both type declarations and method declarations in a single pass.
// This is more efficient than iterating over declarations twice (O(n) instead of O(2n)).
func (s *SuiteScanner) processDeclaration(decl ast.Decl, packageName, sourceFile string) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		if d.Tok == token.TYPE {
			s.processSuiteTypes(d)
		}
	case *ast.FuncDecl:
		s.processTestMethods(d, packageName, sourceFile)
	}
}

// processSuiteTypes finds struct types that embed suite.Suite.
func (s *SuiteScanner) processSuiteTypes(genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		if s.embedsSuiteSuite(structType) {
			s.collector.addSuiteName(typeSpec.Name.Name)
		}
	}
}

// embedsSuiteSuite checks if a struct type embeds suite.Suite.
func (s *SuiteScanner) embedsSuiteSuite(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			continue // Skip named fields
		}
		if selector, ok := field.Type.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok {
				if ident.Name == "suite" && selector.Sel.Name == "Suite" {
					return true
				}
			}
		}
	}
	return false
}

// processTestMethods finds test methods on suite types.
func (s *SuiteScanner) processTestMethods(fn *ast.FuncDecl, packageName, sourceFile string) {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return
	}

	suiteName := s.extractReceiverType(fn.Recv.List[0])
	if suiteName == "" {
		return
	}

	// Filter by target suite if specified
	if s.cfg.TargetSuite != "" && suiteName != s.cfg.TargetSuite {
		return
	}

	s.collector.addSuiteName(suiteName)

	if strings.HasPrefix(fn.Name.Name, "Test") {
		test := TestMethodInfo{
			Name:      fn.Name.Name,
			DependsOn: s.extractDependencies(fn.Doc),
		}
		s.collector.addTest(suiteName, packageName, sourceFile, test)
	}
}

// extractReceiverType extracts the type name from a receiver field (e.g., *SuiteName -> SuiteName).
func (s *SuiteScanner) extractReceiverType(recv *ast.Field) string {
	star, ok := recv.Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

// extractDependencies parses @dependsOn comments from function documentation.
func (s *SuiteScanner) extractDependencies(doc *ast.CommentGroup) []string {
	var dependsOn []string
	if doc != nil {
		for _, comment := range doc.List {
			deps := parseDependsOnComment(comment.Text)
			dependsOn = append(dependsOn, deps...)
		}
	}
	return dependsOn
}

// usesRunSuite checks if a file contains a call to depend.RunSuite.
// Returns true if the file uses depend.RunSuite, false otherwise.
func (s *SuiteScanner) usesRunSuite(f *ast.File) bool {
	uses := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// Look for depend.RunSuite
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "RunSuite" {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "depend" {
						uses = true
						return false
					}
				}
			}
		}
		return true
	})
	return uses
}

// logDiscoveredSuites logs discovered suites in verbose mode.
func (s *SuiteScanner) logDiscoveredSuites() {
	if !s.cfg.Verbose {
		return
	}
	suiteNames := s.collector.suiteNames()
	suiteList := make([]string, 0, len(suiteNames))
	for name := range suiteNames {
		suiteList = append(suiteList, name)
	}
	sort.Strings(suiteList)
	verbosef(s.cfg, fmt.Sprintf("Discovered %d suite(s): %v", len(suiteNames), suiteList))
}

// parseDependsOnComment extracts dependency names from a @dependsOn comment.
// It handles comma-separated lists and filters out empty entries.
//
// Examples:
//   - "// @dependsOn TestA" → ["TestA"]
//   - "// @dependsOn TestA, TestB" → ["TestA", "TestB"]
//   - "// @dependsOn TestA,TestB" → ["TestA", "TestB"] (empty entries filtered)
func parseDependsOnComment(commentText string) []string {
	var deps []string
	text := strings.TrimSpace(strings.TrimPrefix(commentText, "//"))
	if strings.HasPrefix(text, "@dependsOn") {
		depList := strings.TrimSpace(strings.TrimPrefix(text, "@dependsOn"))
		// Split by comma for multiple dependencies
		for _, dep := range strings.Split(depList, ",") {
			dep = strings.TrimSpace(dep)
			if dep != "" {
				deps = append(deps, dep)
			}
		}
	}
	return deps
}

// extractBuildTags extracts build constraint comments from an AST file.
// It looks for //go:build and // +build directives at the top of the file.
// These comments appear before the package declaration and must be preserved
// in generated files to maintain build constraints.
//
// Returns a slice of complete comment lines including the "//" prefix.
// Example output: ["//go:build test", "// +build test"]
func extractBuildTags(f *ast.File) []string {
	var buildTags []string

	// Build tags can appear in any comment group before the package declaration.
	// Scan all comment groups that appear before the package keyword.
	for _, commentGroup := range f.Comments {
		// Stop if we've reached comments after the package declaration
		if commentGroup.Pos() >= f.Package {
			break
		}

		// Check each comment in the group for build tags
		for _, comment := range commentGroup.List {
			text := comment.Text
			// Look for build constraint comments
			if strings.HasPrefix(text, "//go:build") || strings.HasPrefix(text, "// +build") {
				buildTags = append(buildTags, text)
			}
		}
	}

	return buildTags
}
