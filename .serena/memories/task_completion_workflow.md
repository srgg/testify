# Task Completion Workflow

## Before Coding
1. Read `CLAUDE.md` completely
2. Review package documentation in `depend/doc.go`
3. Check existing tests for patterns and conventions
4. Understand if change is breaking or non-breaking
5. Review existing patterns in codebase
6. Understand the context: bug fix, new feature, or documentation

## During Coding
1. **Write tests FIRST** (TDD encouraged)
2. Keep functions small and focused
3. Update documentation inline with code
4. Follow existing code style
5. Only use `Edit` and `Write` tools for file modifications
6. NEVER use `sed`, `awk`, or bash text manipulation
7. Read files with `Read` tool before modifying

## After Coding - Quality Checklist
This checklist MUST be completed before any commit:

### 1. Run Tests
```bash
go test ./...
```
✅ All tests must pass

### 2. Run Vet
```bash
go vet ./...
```
✅ No vet issues allowed

### 3. Format Code
```bash
gofmt -w .
```
✅ All code must be formatted

### 4. Verify Documentation
- [ ] All exports have doc comments
- [ ] Doc comments start with symbol name
- [ ] Examples updated if API changed
- [ ] README reflects current state

### 5. Verify Tests
- [ ] Tests added for new functionality
- [ ] Test coverage > 80%
- [ ] Example tests included where appropriate

### 6. Verify API Stability
- [ ] No breaking changes without version bump
- [ ] Semantic versioning followed
- [ ] CHANGELOG.md updated if needed

### 7. Verify Examples
- [ ] Example code still works
- [ ] Examples are runnable

## Commit Standards
- Use conventional commits: `feat:`, `fix:`, `docs:`, `test:`, etc.
- For breaking changes: `feat!:` or `fix!:` with BREAKING CHANGE in body
- Reference issues where applicable
- Keep commits atomic and logical

## File Modification Rules (CRITICAL)
- ✅ Use `Edit` tool for targeted changes
- ✅ Use `Write` tool for new files only
- ❌ NEVER use `sed`, `awk`, or bash text manipulation
- ❌ NEVER use manual line-by-line edits

## Remember
This is a PUBLIC library. Code quality, API stability, and documentation are paramount. When in doubt, choose simplicity and clarity over cleverness.
