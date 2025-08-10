# ccnewline

A Claude Code hook utility that automatically ensures files end with newline characters when using Edit, MultiEdit, and Write tools.

## Why ccnewline?

Many text files should end with a newline character according to POSIX standards, but editors and tools sometimes create files without proper line endings. Claude Code, when modifying files through Edit, MultiEdit, and Write operations, often creates files without trailing newlines. This can cause issues with:

- Git diffs showing "No newline at end of file" warnings
- Shell tools and scripts expecting proper line endings
- Code linters and formatters
- Concatenating files

Claude Code hooks would require complex shell scripting with `jq` to parse JSON tool outputs and extract file paths. **ccnewline** simplifies this by handling JSON parsing internally, providing a single-purpose tool that automatically adds missing newlines to files modified by Edit, MultiEdit, and Write operations.

## How It Works as a Hook

When you use Claude Code's Edit, MultiEdit, or Write tools:

1. **Claude Code executes the tool** (creates/modifies files)
2. **Hook triggers automatically** with JSON tool output
3. **ccnewline processes** the file paths from the tool output
4. **Missing newlines added** only to files that need them
5. **Results logged** based on your chosen output mode

The process is completely transparent - you don't need to think about it.

## Installation

### Build from source

```bash
git clone <repository-url>
cd ccnewline
go build -o ccnewline
```

## Setup

Add ccnewline to your `.claude/settings.json` as a PostToolUse hook:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|MultiEdit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccnewline"
          }
        ]
      }
    ]
  }
}
```

Use `-d` for debug output or `-s` for silent mode if needed.

## Development

For development and testing:

```bash
# Build and test
go build -o ccnewline
make ci  # Run full test suite
```
