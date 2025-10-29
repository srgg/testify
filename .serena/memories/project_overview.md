# Project Overview

## Purpose
**Depend** is a public Go library that extends `stretchr/testify` with test dependency management capabilities. It allows test suites to declare dependencies between tests, automatically skipping dependent tests when their parent tests fail.

## Key Features
- **Declarative dependency declaration** - Define test dependencies in a simple, declarative way
- **Code generation** - Automatic test registry generation via `dependgen` tool to eliminate/minimize boilerplate code
- Automatically skip tests whose dependencies failed
- Generic-based implementation for type safety
- Integration with testify/suite

## Core Philosophy
The entire idea is to use a **declarative approach** combined with **code generation** to reduce boilerplate code to zero (or at least keep it at minimum). This allows developers to focus on test logic rather than infrastructure.

## Target Users
Public library for Go developers using testify/suite who need to manage test dependencies in their test suites.

## Project Status
- Early development stage
- Core functionality implemented in `deps.go` and `runtime.go`
- Code generator implemented
- Missing: tests, documentation, README
- Planned features (from todo.md): cycle detection, ordering visualization, DOT export, fail-fast mode, multi-suite support

## License
MIT License (Copyright 2025 srgg)
