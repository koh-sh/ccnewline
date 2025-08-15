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

// Logger defines the unified logging interface
type Logger interface {
	// Print outputs a regular message (respects silent mode)
	Print(format string, args ...any)
	// Debug outputs debug information (only when debug mode is enabled)
	Debug(format string, args ...any)
	// DebugSection starts a new debug section with a title
	DebugSection(title string)
	// DebugEnd closes a debug section
	DebugEnd()
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

// VersionHandler handles version display
type VersionHandler struct{}

// ShowVersion displays version information and exits
func (v *VersionHandler) ShowVersion() {
	fmt.Printf("ccnewline %s (Built on %s from Git SHA %s)\n", version, date, commit)
	os.Exit(0)
}

// ArgumentValidator validates command line arguments
type ArgumentValidator struct{}

// ValidateArgs checks that no positional arguments were provided
func (av *ArgumentValidator) ValidateArgs() error {
	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments")
	}
	return nil
}

// FlagParser handles flag parsing
type FlagParser struct {
	versionHandler *VersionHandler
	argValidator   *ArgumentValidator
}

// NewFlagParser creates a new flag parser
func NewFlagParser() *FlagParser {
	return &FlagParser{
		versionHandler: &VersionHandler{},
		argValidator:   &ArgumentValidator{},
	}
}

// Parse parses command line flags and returns configuration
func (fp *FlagParser) Parse() *Config {
	flag.Usage = usage

	var debug, silent, showVersion bool
	defineBoolFlag(&debug, "d", "debug", "Enable debug output")
	defineBoolFlag(&silent, "s", "silent", "Silent mode - no output")
	defineBoolFlag(&showVersion, "v", "version", "Show version information")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fp.versionHandler.ShowVersion()
	}

	// Validate arguments
	if err := fp.argValidator.ValidateArgs(); err != nil {
		flag.Usage()
		os.Exit(1)
	}

	return &Config{Debug: debug, Silent: silent}
}

// parseFlags parses command line flags and returns configuration or exits on error
func parseFlags() *Config {
	parser := NewFlagParser()
	return parser.Parse()
}

// run executes the main processing logic with the given configuration and input
func run(config *Config, input io.Reader) {
	logger := NewConsoleLogger(config)
	filePaths := readFilePathsFromReader(logger, input)
	if len(filePaths) == 0 {
		logger.DebugSection("RESULT")
		logger.Debug("No files to process")
		logger.DebugEnd()
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

	logger.DebugEnd()
}

// ErrorHandler handles error processing and reporting
type ErrorHandler struct {
	ErrorWriter io.Writer
}

// NewErrorHandler creates a new error handler with stderr as default writer
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		ErrorWriter: os.Stderr,
	}
}

// HandleError handles processing errors
func (eh *ErrorHandler) HandleError(logger Logger, filePath string, err error) {
	logger.Debug("Error: %v", err)
	fmt.Fprintf(eh.ErrorWriter, "Error processing %s: %v\n", filePath, err)
}

// ProgressLogger handles progress logging
type ProgressLogger struct{}

// LogProgress logs file processing progress
func (pl *ProgressLogger) LogProgress(logger Logger, filePath string, current, total int) {
	logger.Debug("[%d/%d] Processing: %s", current, total, filePath)
}

// SingleFileProcessor handles single file processing
type SingleFileProcessor struct {
	progressLogger *ProgressLogger
	errorHandler   *ErrorHandler
}

// NewSingleFileProcessor creates a new single file processor
func NewSingleFileProcessor() *SingleFileProcessor {
	return &SingleFileProcessor{
		progressLogger: &ProgressLogger{},
		errorHandler:   NewErrorHandler(),
	}
}

// Process processes a single file with progress logging and error handling
func (sfp *SingleFileProcessor) Process(logger Logger, filePath string, current, total int) {
	sfp.progressLogger.LogProgress(logger, filePath, current, total)
	if err := addNewlineIfNeeded(filePath, logger); err != nil {
		sfp.errorHandler.HandleError(logger, filePath, err)
	}
}

// processSingleFile processes a single file and handles any errors
func processSingleFile(logger Logger, filePath string, current, total int) {
	processor := NewSingleFileProcessor()
	processor.Process(logger, filePath, current, total)
}

// main is the entry point of the ccnewline tool
func main() {
	config := parseFlags()
	run(config, os.Stdin)
}

// Print outputs a regular message (respects silent mode)
func (l *ConsoleLogger) Print(format string, args ...any) {
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

// DebugEnd closes a debug section
func (l *ConsoleLogger) DebugEnd() {
	if l.config.Debug {
		fmt.Printf("└─────────────────────────────────────────────────────────────\n")
	}
}

// DisplayStrategy defines how lines should be displayed
type DisplayStrategy interface {
	Display(logger Logger, lines []string, maxLines int)
}

// TruncatedDisplayStrategy shows lines with truncation for long inputs
type TruncatedDisplayStrategy struct{}

// Display implements DisplayStrategy with truncation logic
func (tds *TruncatedDisplayStrategy) Display(logger Logger, lines []string, maxLines int) {
	if len(lines) <= maxLines {
		for i, line := range lines {
			logger.Debug("  Line %d: %s", i+1, line)
		}
		return
	}

	showFirst := tds.calculateFirstLines(maxLines)

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

// calculateFirstLines determines how many lines to show at the start
func (tds *TruncatedDisplayStrategy) calculateFirstLines(maxLines int) int {
	switch {
	case maxLines >= 5:
		return 2
	default:
		return 1
	}
}

// LineDisplayer handles line display operations
type LineDisplayer struct {
	strategy DisplayStrategy
}

// NewLineDisplayer creates a new line displayer with truncated strategy
func NewLineDisplayer() *LineDisplayer {
	return &LineDisplayer{
		strategy: &TruncatedDisplayStrategy{},
	}
}

// SetStrategy sets the display strategy
func (ld *LineDisplayer) SetStrategy(strategy DisplayStrategy) {
	ld.strategy = strategy
}

// Display displays lines using the configured strategy
func (ld *LineDisplayer) Display(logger Logger, lines []string, maxLines int) {
	ld.strategy.Display(logger, lines, maxLines)
}

// displayLines prints a limited number of lines with truncation for long inputs.
// For inputs longer than maxLines, it shows the first few and last few lines
// with an omission indicator in between.
func displayLines(logger Logger, lines []string, maxLines int) {
	displayer := NewLineDisplayer()
	displayer.Display(logger, lines, maxLines)
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

// TextParser defines the interface for parsing text input
type TextParser interface {
	Parse(inputText string) []string
	CanParse(inputText string) bool
}

// JSONTextParser handles JSON input parsing
type JSONTextParser struct{}

// CanParse checks if the input can be parsed as JSON
func (jtp *JSONTextParser) CanParse(inputText string) bool {
	return len(extractFilePaths(inputText)) > 0
}

// Parse extracts paths from JSON input
func (jtp *JSONTextParser) Parse(inputText string) []string {
	return extractFilePaths(inputText)
}

// PlainTextParser handles plain text input parsing
type PlainTextParser struct{}

// CanParse always returns true as it's the fallback parser
func (ptp *PlainTextParser) CanParse(inputText string) bool {
	return true
}

// Parse extracts paths from plain text input
func (ptp *PlainTextParser) Parse(inputText string) []string {
	lines := strings.Split(inputText, "\n")
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}
	return cleanLines
}

// CompositeTextParser chains multiple parsers
type CompositeTextParser struct {
	parsers []TextParser
}

// NewCompositeTextParser creates a new composite parser with default parsers
func NewCompositeTextParser() *CompositeTextParser {
	return &CompositeTextParser{
		parsers: []TextParser{
			&JSONTextParser{},
			&PlainTextParser{},
		},
	}
}

// AddParser adds a new parser to the chain
func (ctp *CompositeTextParser) AddParser(parser TextParser) {
	ctp.parsers = append(ctp.parsers, parser)
}

// Parse tries each parser in order until one succeeds
func (ctp *CompositeTextParser) Parse(inputText string) []string {
	inputText = strings.TrimSpace(inputText)
	if inputText == "" {
		return nil
	}

	lines := strings.Split(inputText, "\n")
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	if len(cleanLines) == 0 {
		return nil
	}

	jsonText := strings.Join(cleanLines, "\n")
	for _, parser := range ctp.parsers {
		if parser.CanParse(jsonText) {
			if paths := parser.Parse(jsonText); len(paths) > 0 {
				return paths
			}
		}
	}

	return nil
}

// parseFilePathsFromText is a pure function that extracts file paths from input text.
// It attempts JSON parsing first, then falls back to plain text parsing.
func parseFilePathsFromText(inputText string) []string {
	parser := NewCompositeTextParser()
	return parser.Parse(inputText)
}

// InputReader handles reading and parsing input
type InputReader struct {
	inputChecker *InputChecker
	pathParser   *PathParser
}

// NewInputReader creates a new input reader
func NewInputReader() *InputReader {
	return &InputReader{
		inputChecker: &InputChecker{},
		pathParser:   &PathParser{},
	}
}

// InputChecker handles input validation
type InputChecker struct{}

// CheckAvailability checks if input is available from the reader
func (ic *InputChecker) CheckAvailability(logger Logger, input io.Reader) bool {
	return hasInputAvailable(logger, input)
}

// PathParser handles path extraction and parsing
type PathParser struct{}

// Parse extracts paths from input text
func (pp *PathParser) Parse(inputText string) []string {
	return parseFilePathsFromText(inputText)
}

// IsJSON checks if the parsing was done using JSON
func (pp *PathParser) IsJSON(inputText string) bool {
	return extractFilePaths(inputText) != nil
}

// ReadPaths reads and extracts file paths from input
func (ir *InputReader) ReadPaths(logger Logger, input io.Reader) []string {
	if !ir.inputChecker.CheckAvailability(logger, input) {
		return nil
	}

	logger.DebugSection("INPUT PARSING")
	lines := readInputLines(input)

	if len(lines) == 0 {
		logger.Debug("Empty input")
		logger.DebugEnd()
		return nil
	}

	logger.Debug("Input received (%d lines):", len(lines))
	displayLines(logger, lines, 3)

	inputText := strings.Join(lines, "\n")
	paths := ir.pathParser.Parse(inputText)

	if len(paths) > 0 {
		if ir.pathParser.IsJSON(inputText) {
			logger.Debug("JSON parsing successful")
		} else {
			logger.Debug("JSON parsing failed, treating as plain text")
		}
		logger.Debug("Extracted file paths:")
		for i, path := range paths {
			logger.Debug("  [%d] %s", i+1, path)
		}
	}

	logger.DebugEnd()
	return paths
}

// readFilePathsFromReader reads JSON input from the given reader and extracts file paths from
// Claude Code tool outputs. It first attempts JSON parsing to extract paths
// from tool_input fields, falling back to plain text parsing if JSON fails.
func readFilePathsFromReader(logger Logger, input io.Reader) []string {
	reader := NewInputReader()
	return reader.ReadPaths(logger, input)
}

// hasInputAvailable checks if input is available from the reader
func hasInputAvailable(logger Logger, input io.Reader) bool {
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			logger.DebugSection("INPUT PARSING")
			logger.Debug("No stdin input available")
			logger.DebugEnd()
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

// FileProcessor handles file processing operations
type FileProcessor struct {
	validator *FileValidator
	checker   *NewlineChecker
	modifier  *FileModifier
}

// NewFileProcessor creates a new file processor
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{
		validator: &FileValidator{},
		checker:   &NewlineChecker{},
		modifier:  &FileModifier{},
	}
}

// FileValidator handles file validation
type FileValidator struct{}

// ShouldProcess checks if a file should be processed
func (fv *FileValidator) ShouldProcess(filePath string, logger Logger) bool {
	return shouldProcessFile(filePath, logger)
}

// NewlineChecker handles newline checking
type NewlineChecker struct{}

// NeedsNewline checks if a file needs a newline
func (nc *NewlineChecker) NeedsNewline(file *os.File) (bool, error) {
	return checkLastByte(file)
}

// FileModifier handles file modifications
type FileModifier struct{}

// AddNewline adds a newline to a file
func (fm *FileModifier) AddNewline(file *os.File, filePath string, logger Logger) error {
	return addNewlineToFile(file, filePath, logger)
}

// ProcessFile processes a single file for newline addition
func (fp *FileProcessor) ProcessFile(filePath string, logger Logger) error {
	if !fp.validator.ShouldProcess(filePath, logger) {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	needsNewline, err := fp.checker.NeedsNewline(file)
	if err != nil {
		return err
	}

	if needsNewline {
		return fp.modifier.AddNewline(file, filePath, logger)
	}

	logger.Debug("Already ends with newline")
	debugFileContents(logger, filePath)
	return nil
}

// addNewlineIfNeeded checks if a file ends with a newline character and adds
// one if missing. It skips non-existent or empty files and only modifies files
// that don't end with a newline (0x0a byte).
func addNewlineIfNeeded(filePath string, logger Logger) error {
	processor := NewFileProcessor()
	return processor.ProcessFile(filePath, logger)
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
		logger.Print("Added newline to %s\n", filePath)
	}
	return err
}
