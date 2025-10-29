# Changelog

All notable changes to this project will be documented in this file.

This project follows the [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format
and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — Unreleased

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

[0.1.0]: https://github.com/srgg/testify/releases/tag/v0.1.0
