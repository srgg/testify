# CLAUDE.md

## CRITICAL: Session Initialization

**MANDATORY**: At EVERY new Claude Code session start initialize Serena MCP

## Project Overview

Depend is a public Go library providing extensions for `stretchr/testify`. This is a community project that must maintain high code quality, documentation, and
API stability.

## Session Initialization

**At every new Claude Code session:**

1. Read this file completely
2. Review package documentation in `depend/doc.go`
3. Check existing tests for patterns and conventions
4. Understand this is a PUBLIC library - API changes require care

## Project Standards

### Code Quality (Non-Negotiable)

1. **Go Idioms**
    - Follow effective Go and standard Go conventions
    - Use `gofmt` and `go vet` - code must pass both
    - Prefer simple, readable code over clever solutions
    - Keep cyclomatic complexity low

2. **Public API Design**
    - All exported symbols MUST have doc comments
    - Doc comments start with symbol name: `// Dep represents...`
    - Examples in godoc where helpful
    - API stability matters - breaking changes need major version bump
    - Use semantic versioning

3. **Testing**
    - Every exported function/type needs tests
    - Use table-driven tests where appropriate
    - Test coverage should be > 80%
    - Include example tests for documentation
    - Use `testify/assert` for assertions (dogfooding)

4. **Documentation**
    - README.md with clear usage examples
    - Package-level doc.go explaining concepts
    - Runnable examples in `*_test.go` files
    - Keep CHANGELOG.md updated

### File Modification Rules

**CRITICAL**: Use ONLY these tools for modifications:
- ✅ `Edit` - For targeted changes
- ✅ `Write` - For new files only
- ❌ NEVER use `sed`, `awk`, or bash text manipulation
- ❌ NEVER use manual line-by-line edits

**Before ANY modification:**
1. Read the file first with `Read` tool
2. Verify your changes with `go build` and `go test`
3. Run `gofmt -w .` after changes

### Session Workflow

1. **Before coding:**
    - Understand the context (bug fix, new feature, docs)
    - Check if change is breaking or non-breaking
    - Review existing patterns in codebase

2. **During coding:**
    - Write tests FIRST (TDD encouraged)
    - Keep functions small and focused
    - Update documentation inline
    - Follow existing code style

3. **After coding:**
    - Run `go test ./...`
    - Run `go vet ./...`
    - Run `gofmt -w .`
    - Verify examples still work
    - Update README if API changed

### Package-Specific Guidelines

**depend/ package:**
- Core logic in `deps.go` (dependency graph)
- Runtime execution in `runtime.go`
- Must support both suite and non-suite tests
- Performance matters - avoid unnecessary allocations
- Error messages must be clear and actionable

### Commit Standards

- Use conventional commits: `feat:`, `fix:`, `docs:`, `test:`, etc.
- Breaking changes: `feat!:` or `fix!:` with BREAKING CHANGE in body
- Reference issues where applicable
- Keep commits atomic and logical

### What This Project Is NOT

- ❌ Not a place for experimental features
- ❌ Not a dumping ground for all testify utilities
- ❌ Not tied to any specific company/internal code
- ✅ Focused, stable, well-documented testing utilities

## Quality Checklist

Before committing:
- [ ] Code passes `go test ./...`
- [ ] Code passes `go vet ./...`
- [ ] Code is formatted with `gofmt`
- [ ] All exports have doc comments
- [ ] Tests added for new functionality
- [ ] Examples updated if API changed
- [ ] README reflects current state
- [ ] No breaking changes without version bump

## Release Process

### Creating a Release

This project uses **GoReleaser** with GitHub Actions to automate releases.

**Prerequisites:**
- All changes committed and pushed to `main`
- Tests passing (`go test ./...`)
- CHANGELOG.md updated with release notes
- Version follows semantic versioning (MAJOR.MINOR.PATCH)

**Release Steps:**

1. **Create and push a version tag:**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0: Initial public release"
   git push origin v0.1.0
   ```

2. **GitHub Actions automatically:**
   - Runs all tests
   - Builds binaries for multiple platforms (Linux, macOS, Windows on amd64/arm64)
   - Creates GitHub release with binaries and checksums
   - Generates changelog from conventional commits

3. **Verify the release:**
   - Check GitHub releases page: https://github.com/srgg/testify/releases
   - Verify binaries are attached
   - Review auto-generated release notes
   - Test installation: `go install github.com/srgg/testify/depend/cmd/dependgen@v0.1.0`

**Versioning Guidelines:**
- **Patch** (0.0.x): Bug fixes, documentation, no API changes
- **Minor** (0.x.0): New features, backward compatible
- **Major** (x.0.0): Breaking API changes

**What Gets Released:**
- Library: `github.com/srgg/testify/depend` (Go module)
- Tool binaries: `dependgen` (Linux, macOS, Windows)

**Pre-releases:**
Use suffix for beta/RC versions:
```bash
git tag -a v0.2.0-beta.1 -m "Beta release for v0.2.0"
```

**Rollback a Release:**
If you need to delete a bad release:
```bash
git tag -d v0.1.0              # Delete local tag
git push --delete origin v0.1.0  # Delete remote tag
```
Then delete the release from GitHub UI.

### Testing Releases Locally

Before pushing a tag, test the release process:

```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser@latest

# Build and test release (without publishing)
goreleaser release --snapshot --clean

# Check generated binaries in ./dist/
ls -la dist/
```

## Remember

This is a PUBLIC library used by others. Code quality, API stability, and documentation are paramount. When in doubt, choose simplicity and clarity
over cleverness.