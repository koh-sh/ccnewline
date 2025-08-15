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

// Logger defines the interface for logging operations
type Logger interface {
	// Log outputs a regular message (respects silent mode)
	Log(format string, args ...any)
	// Debug outputs debug information (only when debug mode is enabled)
	Debug(format string, args ...any)
	// DebugSection starts a new debug section with a title
	DebugSection(title string)
	// DebugSeparator closes a debug section
	DebugSeparator()
}

// ConsoleLogger implements Logger interface for console output
type ConsoleLogger struct {
	config *Config
}

// NewConsoleLogger creates a new console logger with the given configuration
func NewConsoleLogger(config *Config) Logger {
	return &ConsoleLogger{config: config}
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
	logger := NewConsoleLogger(config)
	filePaths := readFilePathsFromReader(logger, input)
	if len(filePaths) == 0 {
		logger.DebugSection("RESULT")
		logger.Debug("No files to process")
		logger.DebugSeparator()
		return
	}

	processFiles(logger, filePaths)
}

// processFiles handles the processing of multiple files with debug output
func processFiles(logger Logger, filePaths []string) {
	logger.DebugSection("PROCESSING")
	logger.Debug("Total files to process: %d", len(filePaths))

	for i, filePath := range filePaths {
		processSingleFile(logger, filePath, i+1, len(filePaths))
	}

	logger.DebugSeparator()
}

// processSingleFile processes a single file and handles any errors
func processSingleFile(logger Logger, filePath string, current, total int) {
	logger.Debug("[%d/%d] Processing: %s", current, total, filePath)
	if err := addNewlineIfNeeded(filePath, logger); err != nil {
		logger.Debug("Error: %v", err)
		fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", filePath, err)
	}
}

// main is the entry point of the ccnewline tool
func main() {
	config := parseFlags()
	run(config, os.Stdin)
}

// Log outputs a regular message (respects silent mode)
func (l *ConsoleLogger) Log(format string, args ...any) {
	if !l.config.Silent && !l.config.Debug {
		fmt.Printf(format, args...)
	}
}

// Debug outputs debug information (only when debug mode is enabled)
func (l *ConsoleLogger) Debug(format string, args ...any) {
	if l.config.Debug {
		fmt.Printf("│ "+format+"\n", args...)
	}
}

// DebugSection starts a new debug section with a title
func (l *ConsoleLogger) DebugSection(title string) {
	if l.config.Debug {
		fmt.Printf("\n┌─ %s ─────────────────────────────────────────────────────────\n", title)
	}
}

// DebugSeparator closes a debug section
func (l *ConsoleLogger) DebugSeparator() {
	if l.config.Debug {
		fmt.Printf("└─────────────────────────────────────────────────────────────\n")
	}
}

// displayLines prints a limited number of lines with truncation for long inputs.
// For inputs longer than maxLines, it shows the first few and last few lines
// with an omission indicator in between.
func displayLines(logger Logger, lines []string, maxLines int) {
	if len(lines) <= maxLines {
		for i, line := range lines {
			logger.Debug("  Line %d: %s", i+1, line)
		}
		return
	}

	// For 5-line limit, show 2 lines at start; for 3-line limit, show 1 line at start
	showFirst := 1
	if maxLines == 5 {
		showFirst = 2
	}

	for i := 0; i < showFirst && i < len(lines); i++ {
		logger.Debug("  Line %d: %s", i+1, lines[i])
	}

	// Show omission indicator with count of hidden lines
	omitted := len(lines) - maxLines
	logger.Debug("  ... (%d lines omitted) ...", omitted)

	// Show the last few lines after omission
	showLast := maxLines - showFirst
	start := len(lines) - showLast
	for i := start; i < len(lines); i++ {
		logger.Debug("  Line %d: %s", i+1, lines[i])
	}
}

// debugFileContents reads and displays the contents of a file in debug mode,
// showing up to 5 lines with truncation for longer files
func debugFileContents(logger Logger, filePath string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Debug("Failed to read file contents: %v", err)
		return
	}

	logger.Debug("File contents:")
	lines := strings.Split(string(content), "\n")
	displayLines(logger, lines, 5)
}

// parseFilePathsFromText is a pure function that extracts file paths from input text.
// It attempts JSON parsing first, then falls back to plain text parsing.
func parseFilePathsFromText(inputText string) []string {
	inputText = strings.TrimSpace(inputText)
	if inputText == "" {
		return nil
	}

	lines := strings.Split(inputText, "\n")
	// Trim empty lines
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	if len(cleanLines) == 0 {
		return nil
	}

	// Try JSON parsing first
	jsonText := strings.Join(cleanLines, "\n")
	if paths := extractFilePaths(jsonText); len(paths) > 0 {
		return paths
	}

	// Fall back to plain text parsing
	return cleanLines
}

// readFilePathsFromReader reads JSON input from the given reader and extracts file paths from
// Claude Code tool outputs. It first attempts JSON parsing to extract paths
// from tool_input fields, falling back to plain text parsing if JSON fails.
func readFilePathsFromReader(logger Logger, input io.Reader) []string {
	if !hasInputAvailable(logger, input) {
		return nil
	}

	logger.DebugSection("INPUT PARSING")
	lines := readInputLines(input)

	if len(lines) == 0 {
		logger.Debug("Empty input")
		logger.DebugSeparator()
		return nil
	}

	logger.Debug("Input received (%d lines):", len(lines))
	displayLines(logger, lines, 3)

	inputText := strings.Join(lines, "\n")
	paths := parseFilePathsFromText(inputText)

	if len(paths) > 0 {
		// Check if we used JSON parsing or plain text
		if extractFilePaths(inputText) != nil {
			logger.Debug("JSON parsing successful")
		} else {
			logger.Debug("JSON parsing failed, treating as plain text")
		}
		logger.Debug("Extracted file paths:")
		for i, path := range paths {
			logger.Debug("  [%d] %s", i+1, path)
		}
	}

	logger.DebugSeparator()
	return paths
}

// hasInputAvailable checks if input is available from the reader
func hasInputAvailable(logger Logger, input io.Reader) bool {
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			logger.DebugSection("INPUT PARSING")
			logger.Debug("No stdin input available")
			logger.DebugSeparator()
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

// needsNewlineFromContent is a pure function that checks if content needs a trailing newline
func needsNewlineFromContent(content []byte) bool {
	if len(content) == 0 {
		return false // Empty files don't need newlines
	}
	return content[len(content)-1] != newlineByte
}

// fileExists is a pure function that checks if a file exists
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// isFileEmpty is a pure function that checks if a file is empty
func isFileEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false // Non-existent files are not considered empty
	}
	return info.Size() == 0
}

// addNewlineIfNeeded checks if a file ends with a newline character and adds
// one if missing. It skips non-existent or empty files and only modifies files
// that don't end with a newline (0x0a byte).
func addNewlineIfNeeded(filePath string, logger Logger) error {
	if !shouldProcessFile(filePath, logger) {
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
		return addNewlineToFile(file, filePath, logger)
	}

	logger.Debug("Already ends with newline")
	debugFileContents(logger, filePath)
	return nil
}

// shouldProcessFile checks if the file exists and is not empty
func shouldProcessFile(filePath string, logger Logger) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		logger.Debug("File does not exist, skipping")
		return false
	}
	if info.Size() == 0 {
		logger.Debug("File is empty, skipping")
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
func addNewlineToFile(file *os.File, filePath string, logger Logger) error {
	logger.Debug("Adding newline (missing)")
	debugFileContents(logger, filePath)

	_, err := file.Write([]byte{newlineByte})
	if err == nil {
		logger.Debug("Newline added successfully")
		logger.Log("Added newline to %s\n", filePath)
	}
	return err
}
