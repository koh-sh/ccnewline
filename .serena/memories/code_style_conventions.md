# Code Style and Conventions

## Language and Tech Stack
- **Language**: Go 1.24
- **Dependencies**: Standard library only (no external dependencies)
- **Architecture**: Single-file implementation with SOLID compliance

## Code Style
- **Formatter**: gofumpt (via `make fmt`)
- **Linter**: golangci-lint with custom configuration
- **Import organization**: gci formatter for import grouping

## Enabled Linters
- asciicheck - Check for non-ASCII characters
- gocritic - Go code critic
- misspell - Check for commonly misspelled words
- nolintlint - Validate nolint comments
- predeclared - Check for shadowing of predeclared identifiers
- unconvert - Check for unnecessary type conversions

## Naming Conventions
- Variables: camelCase (e.g., `newlineByte`, `maxDisplayLines`)
- Constants: camelCase with descriptive names (e.g., `filePermission`)
- Types: PascalCase for public, camelCase for private (e.g., `config`, `consoleLogger`)
- Functions: camelCase (e.g., `newConsoleLogger`, `processFiles`)

## Design Patterns
- **SOLID Principles**: Architecture follows SOLID compliance
- **Single Responsibility**: Each struct has a focused responsibility
- **Interface Segregation**: Small, focused interfaces (e.g., `logger`, `textParser`, `displayStrategy`)
- **Dependency Injection**: Constructor functions for struct initialization

## File Organization
- All code in `main.go` (~732 lines)
- Logical grouping of related functionality
- Table-driven tests in `main_test.go`

## Comments
- Struct comments explaining purpose
- Field comments for configuration options
- Function comments when behavior isn't obvious from name