# Task Completion Workflow

## Quality Assurance Process
**For any code modification work, always follow this quality assurance process:**

1. **Create todo list**: Use todo list to track work items and ensure no tasks are missed

2. **Continuous quality checks**: Execute the following commands throughout development:
   ```bash
   make ci            # Code quality, linting, formatting, and unit tests
   make blackboxtest  # Integration tests (7 focused test cases)
   make cov           # Test coverage verification
   ```

3. **Test-driven approach**: Maintain or improve test coverage when adding new functionality

4. **Table-driven tests**: When writing tests, use table-driven test format with:
   - No complex functions in test tables
   - No complex if/switch branches in test execution
   - Simple, data-driven test cases
   - If the above requirements cannot be met, consider improving the application code instead

## Pre-commit Requirements
These quality checks must pass before committing any changes:
- `make ci` must pass (includes fmt, modernize-fix, lint, test)
- `make blackboxtest` must pass
- Test coverage should be maintained or improved

## Testing Architecture
- **Unit tests**: Comprehensive table-driven tests in `main_test.go`
- **Integration tests**: 7 minimal tests in `_testscripts/test_functionality.sh`
- **Coverage target**: Currently at 84.0%, aim to maintain or improve

## Build Verification
- `go build -o ccnewline` should complete without errors
- Binary should be functional for hook testing