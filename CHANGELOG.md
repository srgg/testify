# Changelog

All notable changes to this project will be documented in this file.

This project follows the [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format
and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] — 2025-11-03

### Added
- **dependgen**: Human-readable dependency documentation in generated files ([#3](https://github.com/srgg/testify/issues/3))

### Fixed
- **dependgen**: Build tags (`//go:build` and `// +build` directives) are now correctly preserved from source test files to generated `*_depend_test.go` files ([#1](https://github.com/srgg/testify/issues/1))
  - Scans all comment groups before package declaration to find build constraints
  - Handles files with copyright notices or other comments before build tags
  - Ensures generated files maintain the same build constraints as source files

## [0.1.0] — 2024-11-03

Initial public release of `testify/depend`, bringing first-class dependency management
to Go test suites built with `stretchr/testify`.

### Added
- `depend` package for defining and enforcing explicit dependencies between test methods
- `dependgen` CLI for generating dependency configuration based on test annotations
- Lightweight annotation syntax using `@dependsOn` test comments
- Multiple operating modes to fit different workflows:
    - Auto-detect (default `go generate` usage)
    - Explicit suite targeting
    - Multi-suite generation
- Safety features to prevent invalid dependency graphs:
    - Circular dependency protection
    - Missing reference detection
    - Self-dependency checks
- Guaranteed consistent output with atomic file writes
- Cross-platform support (Linux, macOS, Windows — amd64/arm64)
- CI/CD with GitHub Actions, including minimum coverage enforcement (60%)
- GoReleaser configuration for simplified distribution
- Initial documentation and examples

### Notes
- Early-access release — APIs may evolve prior to `v1.0.0`
- Community feedback is encouraged to help shape the final design

[0.1.1]: https://github.com/srgg/testify/releases/tag/v0.1.1
[0.1.0]: https://github.com/srgg/testify/releases/tag/v0.1.0
