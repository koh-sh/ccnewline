# ccnewline

A Claude Code hook utility that automatically ensures files end with newline characters when using Edit, MultiEdit, and Write tools.

## Why ccnewline?

Many text files should end with a newline character according to POSIX standards, but editors and tools sometimes create files without proper line endings. This can cause issues with:

- Git diffs showing "No newline at end of file" warnings
- Shell tools and scripts expecting proper line endings
- Code linters and formatters
- Concatenating files

Traditionally, Claude Code hooks would require complex shell scripting with `jq` to parse JSON tool outputs and extract file paths. **ccnewline** simplifies this by handling JSON parsing internally, providing a single-purpose tool that automatically adds missing newlines to files modified by Edit, MultiEdit, and Write operations.

## Features

- ✅ **Smart detection**: Only adds newlines to files that actually need them
- ✅ **Seamless Claude Code integration**: Designed specifically as a PostToolUse hook
- ✅ **Multiple input formats**: JSON tool input or plain file paths
- ✅ **Three output modes**: Normal, silent, and debug
- ✅ **Zero dependencies**: Uses only Go standard library
- ✅ **Fast and lightweight**: Single binary, minimal resource usage

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

## How It Works as a Hook

When you use Claude Code's Edit, MultiEdit, or Write tools:

1. **Claude Code executes the tool** (creates/modifies files)
2. **Hook triggers automatically** with JSON tool output
3. **ccnewline processes** the file paths from the tool output
4. **Missing newlines added** only to files that need them
5. **Results logged** based on your chosen output mode

The process is completely transparent - you don't need to think about it.

## How It Works

After Claude Code Edit/MultiEdit/Write operations, ccnewline automatically:

1. Receives the tool output via stdin
2. Extracts file paths from the JSON
3. Checks each file's ending
4. Adds newlines only where missing

Normal mode shows "Added newline to [file]" messages when changes are made.

## Hook Command Options

| Flag | Long Form | Description | Best For |
|------|-----------|-------------|----------|
| `-d` | `--debug` | Detailed processing information | Development & troubleshooting |
| `-s` | `--silent` | No output at all | Production environments |
| (none) | | Brief "Added newline to [file]" messages | General use |

## Technical Details

1. **Hook Trigger**: Automatically receives Claude Code tool output as JSON via stdin
2. **Path Extraction**: Parses `tool_input.file_path`, `tool_input.path`, or `tool_input.paths[]`
3. **File Analysis**: Checks each file's last byte to detect missing newlines (0x0a)
4. **Smart Processing**: Only modifies files that actually need newlines
5. **Logging**: Reports actions based on your configured output mode

## Error Handling

ccnewline gracefully handles common scenarios:

- **Non-existent files**: Skipped silently
- **Empty files**: Left unchanged  
- **Read-only files**: Error reported but processing continues
- **Directories**: Skipped with error message
- **Invalid JSON**: Falls back to plain text mode

## Installation Location

Place the binary in a permanent location and use the full path in your Claude Code settings:

```bash
# Example installation
sudo cp ccnewline /usr/local/bin/ccnewline
```

Then use in Claude Code settings:

```json
{
  "hooks": {
    "postToolUse": {
      "command": "/usr/local/bin/ccnewline -d",
      "tools": ["Edit", "MultiEdit", "Write"]
    }
  }
}
```

## Development

For development and testing:

```bash
# Build and test
go build -o ccnewline
make ci  # Run full test suite
```

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
