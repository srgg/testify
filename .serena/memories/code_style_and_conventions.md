# Code Style and Conventions

## Go Idioms (Non-Negotiable)
- Follow Effective Go and standard Go conventions
- Use `gofmt` for formatting - code must pass
- Use `go vet` - code must pass both vet and fmt
- Prefer simple, readable code over clever solutions
- Keep cyclomatic complexity low
- Avoid unnecessary allocations (performance matters)

## Naming Conventions
- Exported symbols MUST have doc comments
- Doc comments start with the symbol name: `// Dep represents...`
- Use clear, descriptive names
- Follow standard Go naming (camelCase for unexported, PascalCase for exported)

## Documentation Requirements
- All exported functions/types need doc comments
- Include examples in godoc where helpful
- Write runnable example tests in `*_test.go` files
- Keep package-level documentation in `doc.go`
- Maintain CHANGELOG.md for version tracking

## Testing Conventions
- Every exported function/type needs tests
- Use table-driven tests where appropriate
- Test coverage MUST be > 80%
- Include example tests for documentation
- Use `testify/assert` for assertions (dogfooding our dependency)
- Write tests FIRST (TDD encouraged)

## API Design Principles
- API stability is critical - this is a PUBLIC library
- Breaking changes require major version bump
- Use semantic versioning
- Error messages must be clear and actionable
- Backward compatibility when possible

## Code Structure
- Keep functions small and focused
- Core logic: `deps.go` (dependency graph management)
- Runtime execution: `runtime.go` 
- Must support both suite and non-suite tests
