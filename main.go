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

// Version information, set by goreleaser during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
  -d, --debug      Enable debug output
  -s, --silent     Silent mode - no output
  -v, --version    Show version information
  -h, --help       Show this help message
`, os.Args[0])
}

// defineBoolFlag defines both short and long form of a boolean flag
func defineBoolFlag(p *bool, short, long string, usage string) {
	flag.BoolVar(p, short, false, usage)
	flag.BoolVar(p, long, false, usage)
}

// parseFlags parses command line flags and returns configuration or exits on error
func parseFlags() *Config {
	flag.Usage = usage

	var debug, silent, showVersion bool
	defineBoolFlag(&debug, "d", "debug", "Enable debug output")
	defineBoolFlag(&silent, "s", "silent", "Silent mode - no output")
	defineBoolFlag(&showVersion, "v", "version", "Show version information")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("ccnewline %s (Built on %s from Git SHA %s)\n", version, date, commit)
		os.Exit(0)
	}

	// Validate that no command line arguments were provided since we only accept stdin
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	return &Config{Debug: debug, Silent: silent}
}

// run executes the main processing logic with the given configuration and input
func run(config *Config, input io.Reader) {
	filePaths := readFilePathsFromReader(config, input)
	if len(filePaths) == 0 {
		config.debugSectionWithInfo("RESULT", "No files to process")
		return
	}

	processFiles(config, filePaths)
}

// processFiles handles the processing of multiple files with debug output
func processFiles(config *Config, filePaths []string) {
	config.debugSection("PROCESSING")
	config.debugInfo("Total files to process: %d", len(filePaths))

	for i, filePath := range filePaths {
		processSingleFile(config, filePath, i+1, len(filePaths))
	}

	config.debugSeparator()
}

// processSingleFile processes a single file and handles any errors
func processSingleFile(config *Config, filePath string, current, total int) {
	config.debugInfo("[%d/%d] Processing: %s", current, total, filePath)
	if err := addNewlineIfNeeded(filePath, config); err != nil {
		config.debugInfo("Error: %v", err)
		fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", filePath, err)
	}
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
	if !hasInputAvailable(config, input) {
		return nil
	}

	config.debugSection("INPUT PARSING")
	lines := readInputLines(input)

	if len(lines) == 0 {
		config.debugInfo("Empty input")
		config.debugSeparator()
		return nil
	}

	config.debugInfo("Input received (%d lines):", len(lines))
	config.displayLines(lines, 3)

	return parseInputLines(config, lines)
}

// hasInputAvailable checks if input is available from the reader
func hasInputAvailable(config *Config, input io.Reader) bool {
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			config.debugSectionWithInfo("INPUT PARSING", "No stdin input available")
			return false
		}
	}
	return true
}

// readInputLines reads and normalizes input lines, trimming empty lines at start and end
func readInputLines(input io.Reader) []string {
	var lines []string
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" || len(lines) > 0 {
			lines = append(lines, line)
		}
	}

	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// parseInputLines attempts JSON parsing first, then falls back to plain text
func parseInputLines(config *Config, lines []string) []string {
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

	return parseAsPlainText(config, lines)
}

// parseAsPlainText treats input as plain text with one file path per line
func parseAsPlainText(config *Config, lines []string) []string {
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
	toolInput := parseJSONToolInput(jsonText)
	if toolInput == nil {
		return nil
	}

	return extractPathsFromToolInput(toolInput)
}

// parseJSONToolInput parses JSON and extracts the tool_input section
func parseJSONToolInput(jsonText string) map[string]any {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		return nil
	}

	toolInput, ok := data["tool_input"].(map[string]any)
	if !ok {
		return nil
	}

	return toolInput
}

// extractPathsFromToolInput collects file paths from various tool_input fields
func extractPathsFromToolInput(toolInput map[string]any) []string {
	var paths []string
	addPath := func(path string) {
		if path != "" {
			paths = append(paths, path)
		}
	}

	// Extract from single path fields
	if path, ok := toolInput["path"].(string); ok {
		addPath(path)
	}
	if filePath, ok := toolInput["file_path"].(string); ok {
		addPath(filePath)
	}

	// Extract from paths array (MultiEdit tool)
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
	if !shouldProcessFile(filePath, config) {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	needsNewline, err := checkLastByte(file)
	if err != nil {
		return err
	}

	if needsNewline {
		return addNewlineToFile(file, filePath, config)
	}

	config.debugInfo("Already ends with newline")
	config.debugFileContents(filePath)
	return nil
}

// shouldProcessFile checks if the file exists and is not empty
func shouldProcessFile(filePath string, config *Config) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		config.debugInfo("File does not exist, skipping")
		return false
	}
	if info.Size() == 0 {
		config.debugInfo("File is empty, skipping")
		return false
	}
	return true
}

// checkLastByte reads the last byte of the file to check if it's a newline
func checkLastByte(file *os.File) (bool, error) {
	_, err := file.Seek(-1, io.SeekEnd)
	if err != nil {
		return false, err
	}

	lastByte := make([]byte, 1)
	_, err = file.Read(lastByte)
	if err != nil {
		return false, err
	}

	return lastByte[0] != newlineByte, nil
}

// addNewlineToFile appends a newline to the file and handles output
func addNewlineToFile(file *os.File, filePath string, config *Config) error {
	config.debugInfo("Adding newline (missing)")
	config.debugFileContents(filePath)

	_, err := file.Write([]byte{newlineByte})
	if err == nil {
		config.debugInfo("Newline added successfully")
		if !config.Debug && !config.Silent {
			fmt.Printf("Added newline to %s\n", filePath)
		}
	}
	return err
}
