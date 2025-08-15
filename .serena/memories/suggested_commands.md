# Suggested Commands for ccnewline Development

## Build Commands
- `go build -o ccnewline` - Build the binary

## Testing Commands
- `make test` - Run unit tests with formatted output
- `go test -v` - Run unit tests with verbose output
- `make cov` - Generate test coverage report (HTML output to cover.html)
- `make blackboxtest` - Run minimal integration tests (7 focused test cases)

## Code Quality Commands
- `make fmt` - Format code with gofumpt
- `make lint` - Run golangci-lint
- `make modernize` - Check for Go modernization opportunities
- `make modernize-fix` - Apply Go modernization fixes
- `make ci` - Full CI pipeline (fmt, modernize-fix, lint, test, coverage, blackboxtest)

## System Commands (Darwin/macOS)
- `git` - Version control
- `ls` - List directory contents
- `grep` or `rg` (ripgrep) - Search in files
- `find` - Find files
- `cd` - Change directory

## Important Notes
- Always run `make ci` before committing changes
- Test coverage is currently at 84.0%
- Integration tests cover Claude Code tool patterns (Edit/MultiEdit/Write)