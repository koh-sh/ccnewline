package toolinput

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/koh-sh/ccnewline/internal/logging"
)

// pathExtractor extracts file paths from various input formats
type pathExtractor struct{}

// newPathExtractor creates a new path extractor
func newPathExtractor() *pathExtractor {
	return &pathExtractor{}
}

// parse extracts file paths from input text (JSON or plain text)
func (pe *pathExtractor) parse(inputText string) []string {
	paths, err := pe.parseJSON(inputText)
	if err == nil {
		return paths
	}
	return pe.parsePlainText(inputText)
}

// isJSON checks if the input text is valid JSON
func (pe *pathExtractor) isJSON(inputText string) bool {
	var jsonObj map[string]any
	return json.Unmarshal([]byte(inputText), &jsonObj) == nil
}

// parseJSON extracts file paths from JSON input
func (pe *pathExtractor) parseJSON(inputText string) ([]string, error) {
	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(inputText), &jsonObj); err != nil {
		return nil, err
	}

	var paths []string

	// Extract from tool_input if present
	if toolInput, exists := jsonObj["tool_input"]; exists {
		if toolInputMap, ok := toolInput.(map[string]any); ok {
			paths = append(paths, pe.extractPathsFromToolInput(toolInputMap)...)
		}
	}

	return paths, nil
}

// extractPathsFromToolInput extracts paths from tool_input object
func (pe *pathExtractor) extractPathsFromToolInput(toolInput map[string]any) []string {
	var paths []string

	// Check for single file path fields
	for _, field := range []string{"path", "file_path"} {
		if value, exists := toolInput[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				paths = append(paths, strValue)
			}
		}
	}

	// Check for paths array
	if pathsValue, exists := toolInput["paths"]; exists {
		if pathsArray, ok := pathsValue.([]any); ok {
			for _, path := range pathsArray {
				if strPath, ok := path.(string); ok && strPath != "" {
					paths = append(paths, strPath)
				}
			}
		}
	}

	return paths
}

// parsePlainText extracts file paths from plain text input
func (pe *pathExtractor) parsePlainText(inputText string) []string {
	lines := strings.Split(inputText, "\n")
	var paths []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	return paths
}

// inputChecker checks input availability
type inputChecker struct{}

// newInputChecker creates a new input checker
func newInputChecker() *inputChecker {
	return &inputChecker{}
}

// checkAvailability checks if input is available for reading
func (ic *inputChecker) checkAvailability(logger logging.Logger, input io.Reader) bool {
	if input == os.Stdin {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			logger.Debug("No stdin input available")
			return false
		}
	}
	return true
}

// inputReader reads and processes input
type inputReader struct {
	pathParser   *pathExtractor
	inputChecker *inputChecker
}

// newInputReader creates a new input reader
func newInputReader() *inputReader {
	return &inputReader{
		pathParser:   newPathExtractor(),
		inputChecker: newInputChecker(),
	}
}

// readPaths reads file paths from input
func (ir *inputReader) readPaths(logger logging.Logger, input io.Reader) []string {
	if !ir.inputChecker.checkAvailability(logger, input) {
		return nil
	}

	lines := readInputLines(input)

	if len(lines) == 0 {
		logger.Debug("Empty input")
		return nil
	}

	logger.Debug(fmt.Sprintf("Input received (%d lines):", len(lines)))
	for i, line := range lines {
		logger.Debug(fmt.Sprintf("  Line %d: %s", i+1, line))
	}

	inputText := strings.Join(lines, "\n")
	paths := ir.pathParser.parse(inputText)

	if len(paths) > 0 {
		if ir.pathParser.isJSON(inputText) {
			logger.Debug("JSON parsing successful")
		} else {
			logger.Debug("Plain text parsing used")
		}
		logger.Debug("Extracted file paths:")
		for i, path := range paths {
			logger.Debug(fmt.Sprintf("  [%d] %s", i+1, path))
		}
	} else {
		logger.Debug("No file paths found")
	}

	return paths
}

// ReadToolInput reads JSON input from the given reader and extracts file paths from tool_input fields
func ReadToolInput(logger logging.Logger, input io.Reader) []string {
	reader := newInputReader()
	return reader.readPaths(logger, input)
}

// ParseToolInput parses tool input JSON and extracts file paths
func ParseToolInput(jsonText string) ([]string, error) {
	extractor := newPathExtractor()
	paths, err := extractor.parseJSON(jsonText)
	if err != nil {
		// Fallback to plain text parsing
		return extractor.parsePlainText(jsonText), nil
	}
	return paths, nil
}

// readInputLines reads and normalizes input lines, trimming empty lines at start and end
func readInputLines(input io.Reader) []string {
	var lines []string
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Trim empty lines from start and end
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	if start >= end {
		return []string{}
	}

	return lines[start:end]
}
