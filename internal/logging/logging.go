package logging

import (
	"fmt"
	"os"
	"strings"
)

// Logger defines the interface for logging operations
type Logger interface {
	Debug(message string)
	Info(message string)
	Error(message string)
	LogFileProcessing(file string, result string)
	ShowProcessingStart(files []string)
	ShowProcessingEnd(totalFiles, processedFiles int)
}

// ConsoleLogger implements Logger for console output
type ConsoleLogger struct {
	debugMode bool
	silent    bool
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(config LoggerConfig) Logger {
	return &ConsoleLogger{
		debugMode: config.IsDebugMode(),
		silent:    config.IsSilent(),
	}
}

// LoggerConfig interface for configuration
type LoggerConfig interface {
	IsDebugMode() bool
	IsSilent() bool
}

// Debug outputs debug messages when debug mode is enabled
func (cl *ConsoleLogger) Debug(message string) {
	if cl.debugMode {
		fmt.Fprintln(os.Stderr, message)
	}
}

// Info outputs informational messages when not in silent mode
func (cl *ConsoleLogger) Info(message string) {
	if !cl.silent {
		fmt.Println(message)
	}
}

// Error outputs error messages to stderr
func (cl *ConsoleLogger) Error(message string) {
	fmt.Fprintln(os.Stderr, message)
}

// LogFileProcessing logs file processing results
func (cl *ConsoleLogger) LogFileProcessing(file string, result string) {
	if cl.debugMode {
		cl.Debug(fmt.Sprintf("  %s: %s", file, result))
	} else if result == "Added newline" && !cl.silent {
		cl.Info(fmt.Sprintf("Added newline to %s", file))
	}
}

// ShowProcessingStart shows the start of processing with debug info
func (cl *ConsoleLogger) ShowProcessingStart(files []string) {
	if !cl.debugMode {
		return
	}

	cl.Debug("┌─────────────────────────────────────────────────────────────────────┐")
	cl.Debug("│ INPUT                                                               │")
	cl.Debug("└─────────────────────────────────────────────────────────────────────┘")

	if len(files) == 0 {
		cl.Debug("No files to process")
		return
	}

	for _, file := range files {
		cl.Debug(fmt.Sprintf("File: %s", file))
	}

	cl.Debug("")
	cl.Debug("┌─────────────────────────────────────────────────────────────────────┐")
	cl.Debug("│ PROCESSING                                                          │")
	cl.Debug("└─────────────────────────────────────────────────────────────────────┘")
}

// ShowProcessingEnd shows the end of processing with debug info
func (cl *ConsoleLogger) ShowProcessingEnd(totalFiles, processedFiles int) {
	if !cl.debugMode {
		return
	}

	cl.Debug("")
	cl.Debug("┌─────────────────────────────────────────────────────────────────────┐")
	cl.Debug("│ SUMMARY                                                             │")
	cl.Debug("└─────────────────────────────────────────────────────────────────────┘")
	cl.Debug(fmt.Sprintf("Total files: %d", totalFiles))
	cl.Debug(fmt.Sprintf("Files processed: %d", processedFiles))
}

// truncateInput truncates input if it's longer than 3 lines for debug display
func truncateInput(input string) string {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) <= 3 {
		return input
	}

	// Show last 3 lines with "..." prefix
	lastThree := lines[len(lines)-3:]
	return "...\n" + strings.Join(lastThree, "\n")
}

// ShowInputDebug shows the input in debug format
func ShowInputDebug(logger Logger, input string) {
	if input == "" {
		return
	}

	truncated := truncateInput(input)
	logger.Debug("Raw input:")
	for line := range strings.SplitSeq(truncated, "\n") {
		if line != "" {
			logger.Debug(fmt.Sprintf("  %s", line))
		}
	}
	logger.Debug("")
}
