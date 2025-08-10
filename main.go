// Package main provides a command-line tool for automatically adding newline characters
// to files processed by Claude Code hooks. It is designed as a PostToolUse hook to
// ensure files modified by Edit, MultiEdit, and Write tools end with proper newlines.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// newlineByte represents the byte value of a newline character (\n)
const newlineByte = 0x0a

// Config holds the configuration options for the tool
type Config struct {
	// Debug enables detailed processing information output
	Debug bool
	// Silent disables all output when processing files
	Silent bool
}

// usage prints the program usage information
func usage() {
	fmt.Fprintf(os.Stderr, `ccnewline - Automatically adds newline characters to files processed by Claude Code hooks
Designed as a PostToolUse hook for Edit, MultiEdit, and Write tools.

Usage: %s [options] < input.json

Options:
`, os.Args[0])
	flag.PrintDefaults()
}

// parseFlags parses command line flags and returns configuration or exits on error
func parseFlags() *Config {
	flag.Usage = usage

	var debug, silent bool
	flag.BoolVar(&debug, "d", false, "Enable debug output")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.BoolVar(&silent, "s", false, "Silent mode - no output")
	flag.BoolVar(&silent, "silent", false, "Silent mode - no output")
	flag.Parse()

	// Validate that no command line arguments were provided since we only accept stdin
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	return &Config{Debug: debug, Silent: silent}
}

// run executes the main processing logic with the given configuration and input
func run(config *Config, input io.Reader) {
	// Read and parse file paths from input (either JSON or plain text)
	filePaths := readFilePathsFromReader(config, input)
	if len(filePaths) == 0 {
		config.debugSectionWithInfo("RESULT", "No files to process")
		return
	}

	config.debugSection("PROCESSING")
	config.debugInfo("Total files to process: %d", len(filePaths))

	// Process each file to add newlines if needed
	for i, filePath := range filePaths {
		config.debugInfo("[%d/%d] Processing: %s", i+1, len(filePaths), filePath)
		if err := addNewlineIfNeeded(filePath, config); err != nil {
			config.debugInfo("Error: %v", err)
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", filePath, err)
		}
	}

	config.debugSeparator()
}

// main is the entry point of the ccnewline tool
func main() {
	config := parseFlags()
	run(config, os.Stdin)
}

// debugSection prints a formatted section header for debug output
func (c *Config) debugSection(title string) {
	if c.Debug {
		fmt.Printf("\n┌─ %s ─────────────────────────────────────────────────────────\n", title)
	}
}

// debugInfo prints formatted debug information with a consistent prefix
func (c *Config) debugInfo(format string, args ...any) {
	if c.Debug {
		fmt.Printf("│ "+format+"\n", args...)
	}
}

// debugSeparator prints a closing line for debug sections
func (c *Config) debugSeparator() {
	if c.Debug {
		fmt.Printf("└─────────────────────────────────────────────────────────────\n")
	}
}

// debugSectionWithInfo is a convenience method that combines section header,
// info message, and separator in a single call
func (c *Config) debugSectionWithInfo(title, message string, args ...any) {
	c.debugSection(title)
	c.debugInfo(message, args...)
	c.debugSeparator()
}

// displayLines prints a limited number of lines with truncation for long inputs.
// For inputs longer than maxLines, it shows the first few and last few lines
// with an omission indicator in between.
func (c *Config) displayLines(lines []string, maxLines int) {
	if len(lines) <= maxLines {
		for i, line := range lines {
			c.debugInfo("  Line %d: %s", i+1, line)
		}
		return
	}

	// For 5-line limit, show 2 lines at start; for 3-line limit, show 1 line at start
	showFirst := 1
	if maxLines == 5 {
		showFirst = 2
	}

	for i := 0; i < showFirst && i < len(lines); i++ {
		c.debugInfo("  Line %d: %s", i+1, lines[i])
	}

	// Show omission indicator with count of hidden lines
	omitted := len(lines) - maxLines
	c.debugInfo("  ... (%d lines omitted) ...", omitted)

	// Show the last few lines after omission
	showLast := maxLines - showFirst
	start := len(lines) - showLast
	for i := start; i < len(lines); i++ {
		c.debugInfo("  Line %d: %s", i+1, lines[i])
	}
}

// debugFileContents reads and displays the contents of a file in debug mode,
// showing up to 5 lines with truncation for longer files
func (c *Config) debugFileContents(filePath string) {
	if !c.Debug {
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		c.debugInfo("Failed to read file contents: %v", err)
		return
	}

	c.debugInfo("File contents:")
	lines := strings.Split(string(content), "\n")
	c.displayLines(lines, 5)
}

// readFilePaths reads JSON input from stdin and extracts file paths from
// Claude Code tool outputs. It first attempts JSON parsing to extract paths
// from tool_input fields, falling back to plain text parsing if JSON fails.
func readFilePaths(config *Config) []string {
	return readFilePathsFromReader(config, os.Stdin)
}

// readFilePathsFromReader reads JSON input from the given reader and extracts file paths from
// Claude Code tool outputs. It first attempts JSON parsing to extract paths
// from tool_input fields, falling back to plain text parsing if JSON fails.
func readFilePathsFromReader(config *Config, input io.Reader) []string {
	// Check if stdin has any data available (not a terminal) - only when using os.Stdin
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			config.debugSectionWithInfo("INPUT PARSING", "No stdin input available")
			return nil
		}
	}

	config.debugSection("INPUT PARSING")

	// Read all lines from input, preserving empty lines after content starts
	var lines []string
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		// Start collecting lines on first non-empty line or if we already have content
		if line != "" || len(lines) > 0 {
			lines = append(lines, line)
		}
	}

	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		config.debugInfo("Empty input")
		config.debugSeparator()
		return nil
	}

	config.debugInfo("Input received (%d lines):", len(lines))
	config.displayLines(lines, 3)

	// Attempt to parse as JSON first (Claude Code tool output)
	inputText := strings.Join(lines, "\n")
	if paths := extractFilePaths(inputText); len(paths) > 0 {
		config.debugInfo("JSON parsing successful")
		config.debugInfo("Extracted file paths:")
		for i, path := range paths {
			config.debugInfo("  [%d] %s", i+1, path)
		}
		config.debugSeparator()
		return paths
	}

	// Fallback: treat input as plain text with one file path per line
	config.debugInfo("JSON parsing failed, treating as plain text")
	var filePaths []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			filePaths = append(filePaths, line)
		}
	}
	config.debugSeparator()
	return filePaths
}

// extractFilePaths parses JSON input and extracts file paths from Claude Code
// tool_input fields. It looks for 'path', 'file_path', and 'paths' fields
// which correspond to different Claude Code tools (Edit, Write, MultiEdit).
func extractFilePaths(jsonText string) []string {
	// Parse JSON and extract top-level structure
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		return nil
	}

	// Extract tool_input section which contains file path information
	toolInput, ok := data["tool_input"].(map[string]any)
	if !ok {
		return nil
	}

	// Collect all file paths from various tool_input fields
	var paths []string
	addPath := func(path string) {
		if path != "" {
			paths = append(paths, path)
		}
	}

	// Extract from "path" field (used by some tools)
	if path, ok := toolInput["path"].(string); ok {
		addPath(path)
	}
	// Extract from "file_path" field (used by Edit, Write tools)
	if filePath, ok := toolInput["file_path"].(string); ok {
		addPath(filePath)
	}
	// Extract from "paths" array (used by MultiEdit tool)
	if pathsArray, ok := toolInput["paths"].([]any); ok {
		for _, p := range pathsArray {
			if pathStr, ok := p.(string); ok {
				addPath(pathStr)
			}
		}
	}

	return paths
}

// addNewlineIfNeeded checks if a file ends with a newline character and adds
// one if missing. It skips non-existent or empty files and only modifies files
// that don't end with a newline (0x0a byte).
func addNewlineIfNeeded(filePath string, config *Config) error {
	// Check if file exists and is not empty
	info, err := os.Stat(filePath)
	if err != nil {
		config.debugInfo("File does not exist, skipping")
		return nil
	}
	if info.Size() == 0 {
		config.debugInfo("File is empty, skipping")
		return nil
	}

	// Open file for reading and writing
	file, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to the last byte of the file to check if it's a newline
	_, err = file.Seek(-1, io.SeekEnd)
	if err != nil {
		return err
	}

	// Read the last byte to check if it's a newline character
	lastByte := make([]byte, 1)
	_, err = file.Read(lastByte)
	if err != nil {
		return err
	}

	// Add newline if the file doesn't end with one
	if lastByte[0] != newlineByte {
		config.debugInfo("Adding newline (missing)")
		config.debugFileContents(filePath)

		// Append newline character to the end of the file
		_, err = file.Write([]byte{newlineByte})
		if err == nil {
			config.debugInfo("Newline added successfully")
			// Output normal message when not in debug or silent mode
			if !config.Debug && !config.Silent {
				fmt.Printf("Added newline to %s\n", filePath)
			}
		}
		return err
	}

	// File already ends with newline, no action needed
	config.debugInfo("Already ends with newline")
	config.debugFileContents(filePath)
	return nil
}
