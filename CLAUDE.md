# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ccnewline is a Claude Code PostToolUse hook that automatically ensures files end with newline characters. It simplifies Claude Code hook development by eliminating the need for complex shell scripting with `jq` to parse JSON tool outputs - instead handling JSON parsing internally to extract file paths and add missing newlines.

## Core Design

Single-file implementation following YAGNI principles:
- All functionality in `main.go` (~271 lines)
- Standard library only (no external dependencies)
- Designed exclusively for Claude Code PostToolUse hook usage
- Simple flag-based configuration: normal, silent (-s), debug (-d)

The tool extracts file paths from `tool_input.path`, `tool_input.file_path`, or `tool_input.paths[]` fields in JSON input.

Debug output is automatically truncated to last 3 lines for inputs longer than 3 lines to avoid cluttering output.

## Development Commands

**Build the binary:**
```bash
go build -o ccnewline
```

**Testing commands:**
```bash
make test          # Run unit tests with formatted output
go test -v         # Run unit tests with verbose output
make cov           # Generate test coverage report (HTML output to cover.html)
make blackboxtest  # Run minimal integration tests (7 focused test cases)
```

**Code quality:**
```bash
make fmt           # Format code with gofumpt
make lint          # Run golangci-lint
make modernize     # Check for Go modernization opportunities
make modernize-fix # Apply Go modernization fixes
make ci            # Full CI pipeline (fmt, modernize-fix, lint, test)
```

## Testing Architecture

The project has focused test coverage (75.0%):

- **Unit tests** (`main_test.go`): 4 core test functions covering JSON parsing, file processing, output modes, and error scenarios
- **Integration tests** (`_testscripts/test_functionality.sh`): 7 minimal tests covering Claude Code tool patterns (Edit/MultiEdit/Write) and output modes (normal/silent/debug)
- **Coverage reporting**: Use `make cov` to generate HTML coverage reports

When making changes, always run `make ci` to ensure all checks pass.

## Claude Code Hook Configuration

This tool is designed exclusively as a Claude Code PostToolUse hook. Configure in `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|MultiEdit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccnewline -d"
          }
        ]
      }
    ]
  }
}
```

**Output modes:**
- Default: Shows "Added newline to [file]" when newlines are added
- `-s`/`--silent`: No output  
- `-d`/`--debug`: Detailed processing information with structured output

The tool processes Edit, MultiEdit, and Write tool outputs automatically, adding trailing newlines only to files that need them.

## Key Architecture Points

1. **JSON Processing**: Handles Claude Code tool output parsing without requiring external `jq` commands
2. **Smart Detection**: Only modifies files missing newlines by checking the last byte (0x0a)  
3. **Hook Integration**: Receives JSON via stdin, extracts file paths, processes files, outputs results
4. **Error Handling**: Gracefully handles non-existent files, empty files, and invalid JSON (falls back to plain text mode)

Do not use this tool for direct bash invocation - it is designed specifically for Claude Code hook automation.
