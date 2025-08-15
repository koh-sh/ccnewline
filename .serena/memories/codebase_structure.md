# Codebase Structure

## Root Directory Files
- `main.go` - Single-file implementation (~732 lines) with all functionality
- `main_test.go` - Comprehensive unit tests with table-driven approach
- `go.mod` - Go module definition (Go 1.24, tools only in dependencies)
- `Makefile` - Build, test, and quality commands
- `README.md` - Project documentation
- `CLAUDE.md` - Claude Code specific instructions and guidance
- `.golangci.yml` - Linter configuration
- `.goreleaser.yaml` - Release automation configuration
- `LICENSE` - Project license

## Test Directory
- `_testscripts/test_functionality.sh` - Integration tests (7 test cases)

## Key Code Components in main.go

### Configuration
- `config` struct - Debug/Silent flags
- Constants for file processing (newlineByte, maxDisplayLines, etc.)

### Core Processing
- `processFiles()` - Main file processing logic
- `processSingleFile()` - Individual file processing
- `addNewlineIfNeeded()` - Core newline addition logic

### Input Processing
- `textParser` interface - JSON/plain text parsing
- `jsonTextParser` - Handles Claude Code tool JSON output
- `plainTextParser` - Fallback for plain text input
- `compositeTextParser` - Combines parsers

### Output/Logging
- `logger` interface - Logging abstraction
- `consoleLogger` - Console output implementation
- `displayStrategy` interface - Output display strategies

### File Operations
- `fileProcessor` - File processing orchestration
- `fileValidator` - File validation logic
- `newlineChecker` - Newline detection
- `fileModifier` - File modification operations

## Architecture Pattern
Single-file design with SOLID principles:
- Interface segregation for testability
- Dependency injection via constructor functions
- Clear separation of concerns within single file