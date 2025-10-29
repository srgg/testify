package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"
)

// TempDirFixture builds test files from templates and manages temporary directories
type TempDirFixture struct {
	t         *testing.T
	files     map[string]string
	changeDir bool
}

// NewTempDirFixture creates a new temporary directory fixture
func NewTempDirFixture(t *testing.T) *TempDirFixture {
	return &TempDirFixture{
		t:         t,
		files:     make(map[string]string),
		changeDir: false,
	}
}

// WithFile adds a file with literal content
func (f *TempDirFixture) WithFile(filename string, content string) *TempDirFixture {
	f.files[filename] = content
	return f
}

// WithChangeDir sets the fixture to change to the created directory after building
func (f *TempDirFixture) WithChangeDir() *TempDirFixture {
	f.changeDir = true
	return f
}

// WithTestFile adds a test file generated from a template
func (f *TempDirFixture) WithTestFile(filename string, templ string, params map[string]any) *TempDirFixture {
	testSuiteTemplate := template.Must(template.New("testSuite").Parse(templ))

	var buf bytes.Buffer
	err := testSuiteTemplate.Execute(&buf, params)
	if err != nil {
		f.t.Fatalf("MUST execute template for %s: %v", filename, err)
	}
	f.files[filename] = buf.String()
	return f
}

// Build creates a temporary directory, writes all test files to it, registers cleanup, and returns the directory path.
// If WithChangeDir() was called, also changes to the created directory and registers cleanup to restore the original directory.
//
// Returns the canonical path with all symlinks resolved (e.g., /var -> /private/var on macOS) to ensure
// path comparisons work correctly when os.Getwd() is used, which always returns resolved paths.
func (f *TempDirFixture) Build() string {
	f.t.Helper()

	dir, err := os.MkdirTemp("", "testify-depend-test-*")
	if err != nil {
		f.t.Fatalf("MUST create temp directory: %v", err)
	}

	// Resolve symlinks for cross-platform compatibility (macOS has /var -> /private/var)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		f.t.Fatalf("MUST resolve symlinks in %s: %v", dir, err)
	}

	f.t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	for filename, content := range f.files {
		filePath := filepath.Join(dir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			f.t.Fatalf("MUST write file %s: %v", filename, err)
		}
	}

	// Change directory if requested
	if f.changeDir {
		oldDir, err := os.Getwd()
		if err != nil {
			f.t.Fatalf("MUST get current directory: %v", err)
		}

		err = os.Chdir(resolvedDir)
		if err != nil {
			f.t.Fatalf("MUST change to directory %s: %v", resolvedDir, err)
		}

		f.t.Cleanup(func() {
			os.Chdir(oldDir)
		})
	}

	return resolvedDir
}

// EnvFixture manages environment variables for tests with automatic cleanup
type EnvFixture struct {
	t                 *testing.T
	cleanupRegistered bool
	savedVars         map[string]struct {
		value   string
		existed bool
	}
}

// NewEnvFixture creates a new environment variable fixture
func NewEnvFixture(t *testing.T) *EnvFixture {
	return &EnvFixture{
		t: t,
		savedVars: make(map[string]struct {
			value   string
			existed bool
		}),
	}
}

// Set sets an environment variable and registers cleanup on first use.
// If value is empty, the variable is unset.
// Automatically restores original state via t.Cleanup().
func (f *EnvFixture) Set(key, value string) *EnvFixture {
	f.t.Helper()

	// Register cleanup on first use
	if !f.cleanupRegistered {
		f.t.Cleanup(f.restore)
		f.cleanupRegistered = true
	}

	// Save the original state if not already saved
	if _, saved := f.savedVars[key]; !saved {
		oldValue, existed := os.LookupEnv(key)
		f.savedVars[key] = struct {
			value   string
			existed bool
		}{oldValue, existed}
	}

	// Set a new value
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}

	return f
}

// restore restores all saved environment variables to their original state
func (f *EnvFixture) restore() {
	f.t.Helper()
	for key, saved := range f.savedVars {
		if saved.existed {
			os.Setenv(key, saved.value)
		} else {
			os.Unsetenv(key)
		}
	}
}
