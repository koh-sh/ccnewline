# Project Overview: ccnewline

## Purpose
ccnewline is a Claude Code PostToolUse hook utility that automatically ensures files end with newline characters when using Edit, MultiEdit, and Write tools. It simplifies hook development by eliminating the need for complex shell scripting with `jq` to parse JSON tool outputs.

## Key Features
- Single-file Go implementation (~732 lines) following YAGNI principles
- Standard library only (no external dependencies)
- Designed exclusively for Claude Code PostToolUse hook usage
- Handles JSON parsing internally to extract file paths from tool outputs
- Optimized performance: handles 1GB+ files in ~0.01 seconds
- Minimal memory footprint - only reads last byte of files

## Architecture
- SOLID-compliant single-file architecture in `main.go`
- Extracts file paths from `tool_input.path`, `tool_input.file_path`, or `tool_input.paths[]` fields
- Three output modes: normal, silent (-s), debug (-d)
- Debug output automatically truncated to last 3 lines for long inputs

## Integration
Configured as a PostToolUse hook in `.claude/settings.json` to automatically process files modified by Edit, MultiEdit, and Write operations.