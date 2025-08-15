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

// config holds the configuration options for the tool
type config struct {
	// Debug enables detailed processing information output
	Debug bool
	// Silent disables all output when processing files
	Silent bool
}

// logger defines the unified logging interface
type logger interface {
	// log outputs a regular message (respects silent mode)
	log(format string, args ...any)
	// debug outputs debug information (only when debug mode is enabled)
	debug(format string, args ...any)
	// debugSection starts a new debug section with a title
	debugSection(title string)
	// debugEnd closes a debug section
	debugEnd()
}

// consoleLogger implements logger interface for console output
type consoleLogger struct {
	config *config
}

// newConsoleLogger creates a new console logger with the given configuration
func newConsoleLogger(config *config) logger {
	return &consoleLogger{config: config}
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

// versionHandler handles version display
type versionHandler struct{}

// ShowVersion displays version information and exits
func (v *versionHandler) showVersion() {
	fmt.Printf("ccnewline %s (Built on %s from Git SHA %s)\n", version, date, commit)
	os.Exit(0)
}

// argumentValidator validates command line arguments
type argumentValidator struct{}

// ValidateArgs checks that no positional arguments were provided
func (av *argumentValidator) validateArgs() error {
	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments")
	}
	return nil
}

// flagParser handles flag parsing
type flagParser struct {
	versionHandler *versionHandler
	argValidator   *argumentValidator
}

// NewflagParser creates a new flag parser
func newFlagParser() *flagParser {
	return &flagParser{
		versionHandler: &versionHandler{},
		argValidator:   &argumentValidator{},
	}
}

// Parse parses command line flags and returns configuration
func (fp *flagParser) parse() *config {
	flag.Usage = usage

	var debug, silent, showVersion bool
	defineBoolFlag(&debug, "d", "debug", "Enable debug output")
	defineBoolFlag(&silent, "s", "silent", "Silent mode - no output")
	defineBoolFlag(&showVersion, "v", "version", "Show version information")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fp.versionHandler.showVersion()
	}

	// Validate arguments
	if err := fp.argValidator.validateArgs(); err != nil {
		flag.Usage()
		os.Exit(1)
	}

	return &config{Debug: debug, Silent: silent}
}

// parseFlags parses command line flags and returns configuration or exits on error
func parseFlags() *config {
	parser := newFlagParser()
	return parser.parse()
}

// run executes the main processing logic with the given configuration and input
func run(config *config, input io.Reader) {
	logger := newConsoleLogger(config)
	filePaths := readFilePathsFromReader(logger, input)
	if len(filePaths) == 0 {
		logger.debugSection("RESULT")
		logger.debug("No files to process")
		logger.debugEnd()
		return
	}

	processFiles(logger, filePaths)
}

// processFiles handles the processing of multiple files with debug output
func processFiles(logger logger, filePaths []string) {
	logger.debugSection("PROCESSING")
	logger.debug("Total files to process: %d", len(filePaths))

	for i, filePath := range filePaths {
		processSingleFile(logger, filePath, i+1, len(filePaths))
	}

	logger.debugEnd()
}

// errorHandler handles error processing and reporting
type errorHandler struct {
	ErrorWriter io.Writer
}

// NewerrorHandler creates a new error handler with stderr as default writer
func newErrorHandler() *errorHandler {
	return &errorHandler{
		ErrorWriter: os.Stderr,
	}
}

// HandleError handles processing errors
func (eh *errorHandler) handleError(logger logger, filePath string, err error) {
	logger.debug("Error: %v", err)
	fmt.Fprintf(eh.ErrorWriter, "Error processing %s: %v\n", filePath, err)
}

// progressLogger handles progress logging
type progressLogger struct{}

// LogProgress logs file processing progress
func (pl *progressLogger) logProgress(logger logger, filePath string, current, total int) {
	logger.debug("[%d/%d] Processing: %s", current, total, filePath)
}

// singleFileProcessor handles single file processing
type singleFileProcessor struct {
	progressLogger *progressLogger
	errorHandler   *errorHandler
}

// NewsingleFileProcessor creates a new single file processor
func newSingleFileProcessor() *singleFileProcessor {
	return &singleFileProcessor{
		progressLogger: &progressLogger{},
		errorHandler:   newErrorHandler(),
	}
}

// Process processes a single file with progress logging and error handling
func (sfp *singleFileProcessor) process(logger logger, filePath string, current, total int) {
	sfp.progressLogger.logProgress(logger, filePath, current, total)
	if err := addNewlineIfNeeded(filePath, logger); err != nil {
		sfp.errorHandler.handleError(logger, filePath, err)
	}
}

// processSingleFile processes a single file and handles any errors
func processSingleFile(logger logger, filePath string, current, total int) {
	processor := newSingleFileProcessor()
	processor.process(logger, filePath, current, total)
}

// main is the entry point of the ccnewline tool
func main() {
	config := parseFlags()
	run(config, os.Stdin)
}

// log outputs a regular message (respects silent mode)
func (l *consoleLogger) log(format string, args ...any) {
	if !l.config.Silent && !l.config.Debug {
		fmt.Printf(format, args...)
	}
}

// debug outputs debug information (only when debug mode is enabled)
func (l *consoleLogger) debug(format string, args ...any) {
	if l.config.Debug {
		fmt.Printf("│ "+format+"\n", args...)
	}
}

// debugSection starts a new debug section with a title
func (l *consoleLogger) debugSection(title string) {
	if l.config.Debug {
		fmt.Printf("\n┌─ %s ─────────────────────────────────────────────────────────\n", title)
	}
}

// debugEnd closes a debug section
func (l *consoleLogger) debugEnd() {
	if l.config.Debug {
		fmt.Printf("└─────────────────────────────────────────────────────────────\n")
	}
}

// displayStrategy defines how lines should be displayed
type displayStrategy interface {
	display(logger logger, lines []string, maxLines int)
}

// truncatedDisplayStrategy shows lines with truncation for long inputs
type truncatedDisplayStrategy struct{}

// Display implements displayStrategy with truncation logic
func (tds *truncatedDisplayStrategy) display(logger logger, lines []string, maxLines int) {
	if len(lines) <= maxLines {
		for i, line := range lines {
			logger.debug("  Line %d: %s", i+1, line)
		}
		return
	}

	showFirst := tds.calculateFirstLines(maxLines)

	for i := 0; i < showFirst && i < len(lines); i++ {
		logger.debug("  Line %d: %s", i+1, lines[i])
	}

	// Show omission indicator with count of hidden lines
	omitted := len(lines) - maxLines
	logger.debug("  ... (%d lines omitted) ...", omitted)

	// Show the last few lines after omission
	showLast := maxLines - showFirst
	start := len(lines) - showLast
	for i := start; i < len(lines); i++ {
		logger.debug("  Line %d: %s", i+1, lines[i])
	}
}

// calculateFirstLines determines how many lines to show at the start
func (tds *truncatedDisplayStrategy) calculateFirstLines(maxLines int) int {
	switch {
	case maxLines >= 5:
		return 2
	default:
		return 1
	}
}

// lineDisplayer handles line display operations
type lineDisplayer struct {
	strategy displayStrategy
}

// NewlineDisplayer creates a new line displayer with truncated strategy
func newLineDisplayer() *lineDisplayer {
	return &lineDisplayer{
		strategy: &truncatedDisplayStrategy{},
	}
}

// SetStrategy sets the display strategy
func (ld *lineDisplayer) setStrategy(strategy displayStrategy) {
	ld.strategy = strategy
}

// Display displays lines using the configured strategy
func (ld *lineDisplayer) display(logger logger, lines []string, maxLines int) {
	ld.strategy.display(logger, lines, maxLines)
}

// displayLines prints a limited number of lines with truncation for long inputs.
// For inputs longer than maxLines, it shows the first few and last few lines
// with an omission indicator in between.
func displayLines(logger logger, lines []string, maxLines int) {
	displayer := newLineDisplayer()
	displayer.display(logger, lines, maxLines)
}

// debugFileContents reads and displays the contents of a file in debug mode,
// showing up to 5 lines with truncation for longer files
func debugFileContents(logger logger, filePath string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.debug("Failed to read file contents: %v", err)
		return
	}

	logger.debug("File contents:")
	lines := strings.Split(string(content), "\n")
	displayLines(logger, lines, 5)
}

// textParser defines the interface for parsing text input
type textParser interface {
	parse(inputText string) []string
	canParse(inputText string) bool
}

// jsonTextParser handles JSON input parsing
type jsonTextParser struct{}

// CanParse checks if the input can be parsed as JSON
func (jtp *jsonTextParser) canParse(inputText string) bool {
	return len(extractFilePaths(inputText)) > 0
}

// Parse extracts paths from JSON input
func (jtp *jsonTextParser) parse(inputText string) []string {
	return extractFilePaths(inputText)
}

// plainTextParser handles plain text input parsing
type plainTextParser struct{}

// CanParse always returns true as it's the fallback parser
func (ptp *plainTextParser) canParse(inputText string) bool {
	return true
}

// Parse extracts paths from plain text input
func (ptp *plainTextParser) parse(inputText string) []string {
	lines := strings.Split(inputText, "\n")
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}
	return cleanLines
}

// compositeTextParser chains multiple parsers
type compositeTextParser struct {
	parsers []textParser
}

// newCompositeTextParser creates a new composite parser with default parsers
func newCompositeTextParser() *compositeTextParser {
	return &compositeTextParser{
		parsers: []textParser{
			&jsonTextParser{},
			&plainTextParser{},
		},
	}
}

// AddParser adds a new parser to the chain
func (ctp *compositeTextParser) addParser(parser textParser) {
	ctp.parsers = append(ctp.parsers, parser)
}

// Parse tries each parser in order until one succeeds
func (ctp *compositeTextParser) parse(inputText string) []string {
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
		if parser.canParse(jsonText) {
			if paths := parser.parse(jsonText); len(paths) > 0 {
				return paths
			}
		}
	}

	return nil
}

// parseFilePathsFromText is a pure function that extracts file paths from input text.
// It attempts JSON parsing first, then falls back to plain text parsing.
func parseFilePathsFromText(inputText string) []string {
	parser := newCompositeTextParser()
	return parser.parse(inputText)
}

// inputReader handles reading and parsing input
type inputReader struct {
	inputChecker *inputChecker
	pathParser   *pathParser
}

// NewinputReader creates a new input reader
func newInputReader() *inputReader {
	return &inputReader{
		inputChecker: &inputChecker{},
		pathParser:   &pathParser{},
	}
}

// inputChecker handles input validation
type inputChecker struct{}

// CheckAvailability checks if input is available from the reader
func (ic *inputChecker) checkAvailability(logger logger, input io.Reader) bool {
	return hasInputAvailable(logger, input)
}

// pathParser handles path extraction and parsing
type pathParser struct{}

// Parse extracts paths from input text
func (pp *pathParser) parse(inputText string) []string {
	return parseFilePathsFromText(inputText)
}

// IsJSON checks if the parsing was done using JSON
func (pp *pathParser) isJSON(inputText string) bool {
	return extractFilePaths(inputText) != nil
}

// ReadPaths reads and extracts file paths from input
func (ir *inputReader) readPaths(logger logger, input io.Reader) []string {
	if !ir.inputChecker.checkAvailability(logger, input) {
		return nil
	}

	logger.debugSection("INPUT PARSING")
	lines := readInputLines(input)

	if len(lines) == 0 {
		logger.debug("Empty input")
		logger.debugEnd()
		return nil
	}

	logger.debug("Input received (%d lines):", len(lines))
	displayLines(logger, lines, 3)

	inputText := strings.Join(lines, "\n")
	paths := ir.pathParser.parse(inputText)

	if len(paths) > 0 {
		if ir.pathParser.isJSON(inputText) {
			logger.debug("JSON parsing successful")
		} else {
			logger.debug("JSON parsing failed, treating as plain text")
		}
		logger.debug("Extracted file paths:")
		for i, path := range paths {
			logger.debug("  [%d] %s", i+1, path)
		}
	}

	logger.debugEnd()
	return paths
}

// readFilePathsFromReader reads JSON input from the given reader and extracts file paths from
// Claude Code tool outputs. It first attempts JSON parsing to extract paths
// from tool_input fields, falling back to plain text parsing if JSON fails.
func readFilePathsFromReader(logger logger, input io.Reader) []string {
	reader := newInputReader()
	return reader.readPaths(logger, input)
}

// hasInputAvailable checks if input is available from the reader
func hasInputAvailable(logger logger, input io.Reader) bool {
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			logger.debugSection("INPUT PARSING")
			logger.debug("No stdin input available")
			logger.debugEnd()
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

// fileProcessor handles file processing operations
type fileProcessor struct {
	validator *fileValidator
	checker   *newlineChecker
	modifier  *fileModifier
}

// NewfileProcessor creates a new file processor
func newFileProcessor() *fileProcessor {
	return &fileProcessor{
		validator: &fileValidator{},
		checker:   &newlineChecker{},
		modifier:  &fileModifier{},
	}
}

// fileValidator handles file validation
type fileValidator struct{}

// ShouldProcess checks if a file should be processed
func (fv *fileValidator) shouldProcess(filePath string, logger logger) bool {
	return shouldProcessFile(filePath, logger)
}

// newlineChecker handles newline checking
type newlineChecker struct{}

// NeedsNewline checks if a file needs a newline
func (nc *newlineChecker) needsNewline(file *os.File) (bool, error) {
	return checkLastByte(file)
}

// fileModifier handles file modifications
type fileModifier struct{}

// AddNewline adds a newline to a file
func (fm *fileModifier) addNewline(file *os.File, filePath string, logger logger) error {
	return addNewlineToFile(file, filePath, logger)
}

// ProcessFile processes a single file for newline addition
func (fp *fileProcessor) processFile(filePath string, logger logger) error {
	if !fp.validator.shouldProcess(filePath, logger) {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	needsNewline, err := fp.checker.needsNewline(file)
	if err != nil {
		return err
	}

	if needsNewline {
		return fp.modifier.addNewline(file, filePath, logger)
	}

	logger.debug("Already ends with newline")
	debugFileContents(logger, filePath)
	return nil
}

// addNewlineIfNeeded checks if a file ends with a newline character and adds
// one if missing. It skips non-existent or empty files and only modifies files
// that don't end with a newline (0x0a byte).
func addNewlineIfNeeded(filePath string, logger logger) error {
	processor := newFileProcessor()
	return processor.processFile(filePath, logger)
}

// shouldProcessFile checks if the file exists and is not empty
func shouldProcessFile(filePath string, logger logger) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		logger.debug("File does not exist, skipping")
		return false
	}
	if info.Size() == 0 {
		logger.debug("File is empty, skipping")
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
func addNewlineToFile(file *os.File, filePath string, logger logger) error {
	logger.debug("Adding newline (missing)")
	debugFileContents(logger, filePath)

	_, err := file.Write([]byte{newlineByte})
	if err == nil {
		logger.debug("Newline added successfully")
		logger.log("Added newline to %s\n", filePath)
	}
	return err
}
