// Package processing provides core file processing functionality for ccnewline.
// It handles file operations, filtering, validation, and newline modification.
package processing

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/koh-sh/ccnewline/internal/cli"
	"github.com/koh-sh/ccnewline/internal/logging"
	"github.com/koh-sh/ccnewline/internal/toolinput"
)

// Constants for file processing
const (
	// newlineByte represents the byte value of a newline character (\n)
	newlineByte = 0x0a

	// filePermission is the default file permission for opening files
	filePermission = 0o644
)

// patternMatcher defines the interface for pattern matching
type patternMatcher interface {
	matches(path string) bool
}

// globPatternMatcher implements pattern matching using glob patterns
type globPatternMatcher struct {
	patterns []string
}

// newGlobPatternMatcher creates a new glob pattern matcher
func newGlobPatternMatcher(patterns []string) *globPatternMatcher {
	return &globPatternMatcher{patterns: patterns}
}

// matches checks if the given path matches any of the patterns
func (gpm *globPatternMatcher) matches(path string) bool {
	if len(gpm.patterns) == 0 {
		return false
	}

	for _, pattern := range gpm.patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// fileFilter handles file filtering based on include/exclude patterns
type fileFilter struct {
	excludeMatcher patternMatcher
	includeMatcher patternMatcher
}

// newFileFilter creates a new file filter with the given configuration
func newFileFilter(config *cli.Config) *fileFilter {
	return &fileFilter{
		excludeMatcher: newGlobPatternMatcher(config.Exclude),
		includeMatcher: newGlobPatternMatcher(config.Include),
	}
}

// shouldProcess determines if a file should be processed based on filters
func (ff *fileFilter) shouldProcess(filePath string) bool {
	// If include patterns are specified, file must match at least one
	if len(ff.includeMatcher.(*globPatternMatcher).patterns) > 0 {
		if !ff.includeMatcher.matches(filePath) {
			return false
		}
	}

	// If exclude patterns are specified, file must not match any
	if ff.excludeMatcher.matches(filePath) {
		return false
	}

	return true
}

// errorHandler handles error processing and reporting
type errorHandler struct {
	ErrorWriter io.Writer
}

// newErrorHandler creates a new error handler
func newErrorHandler() *errorHandler {
	return &errorHandler{
		ErrorWriter: os.Stderr,
	}
}

// handleError logs and reports errors
func (eh *errorHandler) handleError(logger logging.Logger, filePath string, err error) {
	logger.Debug(fmt.Sprintf("Error: %v", err))
	fmt.Fprintf(eh.ErrorWriter, "Error processing %s: %v\n", filePath, err)
}

// progressLogger handles progress reporting during file processing
type progressLogger struct{}

// logProgress reports processing progress
func (pl *progressLogger) logProgress(logger logging.Logger, processed, total int, currentFile string) {
	logger.Debug(fmt.Sprintf("  [%d/%d] Processing: %s", processed, total, currentFile))
}

// singleFileProcessor handles processing of individual files
type singleFileProcessor struct {
	logger       logging.Logger
	errorHandler *errorHandler
	progress     *progressLogger
}

// newSingleFileProcessor creates a new single file processor
func newSingleFileProcessor(logger logging.Logger) *singleFileProcessor {
	return &singleFileProcessor{
		logger:       logger,
		errorHandler: newErrorHandler(),
		progress:     &progressLogger{},
	}
}

// process handles the processing of a single file
func (sfp *singleFileProcessor) process(filePath string, processed, total int) {
	sfp.progress.logProgress(sfp.logger, processed, total, filePath)

	if err := processSingleFile(sfp.logger, filePath); err != nil {
		sfp.errorHandler.handleError(sfp.logger, filePath, err)
	}
}

// fileProcessor handles the main file processing logic
type fileProcessor struct {
	validator *fileValidator
	checker   *newlineChecker
	modifier  *fileModifier
}

// newFileProcessor creates a new file processor
func newFileProcessor() *fileProcessor {
	return &fileProcessor{
		validator: &fileValidator{},
		checker:   &newlineChecker{},
		modifier:  &fileModifier{},
	}
}

// fileValidator validates files before processing
type fileValidator struct{}

// newlineChecker checks if files need newlines
type newlineChecker struct{}

// fileModifier modifies files by adding newlines
type fileModifier struct{}

// processFile processes a single file for newline addition
func (fp *fileProcessor) processFile(logger logging.Logger, filePath string) error {
	return addNewlineIfNeeded(logger, filePath)
}

// ProcessFiles processes multiple files, adding newlines where needed
func ProcessFiles(logger logging.Logger, filePaths []string, filter *fileFilter) int {
	processor := newSingleFileProcessor(logger)
	processedCount := 0

	for _, filePath := range filePaths {
		if !filter.shouldProcess(filePath) {
			logger.Debug(fmt.Sprintf("Skipping %s (filtered)", filePath))
			continue
		}

		processedCount++
		processor.process(filePath, processedCount, len(filePaths))
	}

	return processedCount
}

// Run executes the main processing logic with the given configuration and input
func Run(config *cli.Config, logger logging.Logger, input io.Reader) {
	filePaths := toolinput.ReadToolInput(logger, input)
	logger.ShowProcessingStart(filePaths)

	if len(filePaths) == 0 {
		logger.ShowProcessingEnd(0, 0)
		return
	}

	filter := newFileFilter(config)
	processedCount := ProcessFiles(logger, filePaths, filter)
	logger.ShowProcessingEnd(len(filePaths), processedCount)
}

// processSingleFile processes a single file, adding a newline if needed
func processSingleFile(logger logging.Logger, filePath string) error {
	processor := newFileProcessor()
	return processor.processFile(logger, filePath)
}

// addNewlineIfNeeded adds a newline to a file if it doesn't already end with one
func addNewlineIfNeeded(logger logging.Logger, filePath string) error {
	if !shouldProcessFile(filePath) {
		logger.Debug("│ File does not exist, skipping")
		return nil
	}

	needsNewline, err := checkLastByte(filePath)
	if err != nil {
		return fmt.Errorf("failed to check file: %w", err)
	}

	if !needsNewline {
		logger.Debug("│ Already ends with newline")
		return nil
	}

	logger.Debug("│ Adding newline (missing)")

	if err := addNewlineToFile(filePath); err != nil {
		return fmt.Errorf("failed to add newline: %w", err)
	}

	logger.Debug("│ Newline added successfully")
	logger.Info(fmt.Sprintf("Added newline to %s", filePath))
	return nil
}

// shouldProcessFile checks if a file exists and is not empty
func shouldProcessFile(filePath string) bool {
	return fileExists(filePath) && !isFileEmpty(filePath)
}

// checkLastByte checks if a file ends with a newline character
func checkLastByte(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Seek to the last byte
	if _, err := file.Seek(-1, io.SeekEnd); err != nil {
		// If seek fails, the file might be empty
		return true, nil // Empty files need newlines
	}

	// Read the last byte
	lastByte := make([]byte, 1)
	if _, err := file.Read(lastByte); err != nil {
		return false, err
	}

	// Check if it's a newline
	return lastByte[0] != newlineByte, nil
}

// addNewlineToFile appends a newline character to the end of a file
func addNewlineToFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, filePermission)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte{newlineByte})
	return err
}

// needsNewlineFromContent checks if content needs a newline
func needsNewlineFromContent(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	return content[len(content)-1] != newlineByte
}

// fileExists checks if a file exists
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// isFileEmpty checks if a file is empty
func isFileEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return true
	}
	return info.Size() == 0
}
