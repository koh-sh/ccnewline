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

const newlineByte = 0x0a

type Config struct {
	Debug  bool
	Silent bool
}

func main() {
	var debug, silent bool
	flag.BoolVar(&debug, "d", false, "Enable debug output")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.BoolVar(&silent, "s", false, "Silent mode - no output")
	flag.BoolVar(&silent, "silent", false, "Silent mode - no output")
	flag.Parse()

	config := &Config{Debug: debug, Silent: silent}

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: This tool only accepts JSON input from stdin, not command line arguments\n")
		os.Exit(1)
	}

	filePaths := readFilePaths(config)
	if len(filePaths) == 0 {
		config.debugSectionWithInfo("RESULT", "No files to process")
		return
	}

	config.debugSection("PROCESSING")
	config.debugInfo("Total files to process: %d", len(filePaths))

	for i, filePath := range filePaths {
		config.debugInfo("[%d/%d] Processing: %s", i+1, len(filePaths), filePath)
		if err := addNewlineIfNeeded(filePath, config); err != nil {
			config.debugInfo("Error: %v", err)
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", filePath, err)
		}
	}

	config.debugSeparator()
}

func (c *Config) debugSection(title string) {
	if c.Debug {
		fmt.Printf("\n┌─ %s ─────────────────────────────────────────────────────────\n", title)
	}
}

func (c *Config) debugInfo(format string, args ...any) {
	if c.Debug {
		fmt.Printf("│ "+format+"\n", args...)
	}
}

func (c *Config) debugSeparator() {
	if c.Debug {
		fmt.Printf("└─────────────────────────────────────────────────────────────\n")
	}
}

func (c *Config) debugSectionWithInfo(title, message string, args ...any) {
	c.debugSection(title)
	c.debugInfo(message, args...)
	c.debugSeparator()
}

func displayLines(config *Config, lines []string) {
	if len(lines) > 3 {
		config.debugInfo("  Line 1: %s", lines[0])
		config.debugInfo("  ... (%d lines omitted) ...", len(lines)-3)
		for i := len(lines) - 2; i < len(lines); i++ {
			if i >= 0 {
				config.debugInfo("  Line %d: %s", i+1, lines[i])
			}
		}
	} else {
		for i, line := range lines {
			config.debugInfo("  Line %d: %s", i+1, line)
		}
	}
}

func (c *Config) debugFileContents(filePath string) {
	if !c.Debug {
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		c.debugInfo("Failed to read file contents: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	c.debugInfo("File contents:")

	if len(lines) > 5 {
		// Show first 2 and last 3 lines for longer files
		c.debugInfo("  Line 1: %s", lines[0])
		if len(lines) > 1 {
			c.debugInfo("  Line 2: %s", lines[1])
		}
		c.debugInfo("  ... (%d lines omitted) ...", len(lines)-5)
		for i := len(lines) - 3; i < len(lines); i++ {
			if i >= 0 && i < len(lines) {
				c.debugInfo("  Line %d: %s", i+1, lines[i])
			}
		}
	} else {
		// Show all lines for shorter files
		for i, line := range lines {
			c.debugInfo("  Line %d: %s", i+1, line)
		}
	}
}

func readFilePaths(config *Config) []string {
	var filePaths []string

	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		config.debugSectionWithInfo("INPUT PARSING", "No stdin input available")
		return filePaths
	}

	config.debugSection("INPUT PARSING")

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" || len(lines) > 0 { // Keep empty lines in middle, skip leading empty lines
			lines = append(lines, line)
		}
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		config.debugInfo("Empty stdin input")
		config.debugSeparator()
		return filePaths
	}

	// Show input in a structured way
	config.debugInfo("Input received (%d lines):", len(lines))
	displayLines(config, lines)

	// Try JSON parsing first
	inputText := strings.Join(lines, "\n")
	paths := extractFilePaths(inputText)
	if len(paths) > 0 {
		config.debugInfo("JSON parsing successful")
		config.debugInfo("Extracted file paths:")
		for i, path := range paths {
			config.debugInfo("  [%d] %s", i+1, path)
		}
		filePaths = append(filePaths, paths...)
	} else {
		config.debugInfo("JSON parsing failed, treating as plain text")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				filePaths = append(filePaths, line)
			}
		}
	}

	config.debugSeparator()
	return filePaths
}

func extractFilePaths(jsonText string) []string {
	var paths []string
	var data map[string]any

	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		return nil
	}

	toolInput, ok := data["tool_input"].(map[string]any)
	if !ok {
		return nil
	}

	if path, ok := toolInput["path"].(string); ok && path != "" {
		paths = append(paths, path)
	}

	if filePath, ok := toolInput["file_path"].(string); ok && filePath != "" {
		paths = append(paths, filePath)
	}

	if pathsArray, ok := toolInput["paths"].([]any); ok {
		for _, p := range pathsArray {
			if pathStr, ok := p.(string); ok && pathStr != "" {
				paths = append(paths, pathStr)
			}
		}
	}

	return paths
}

func addNewlineIfNeeded(filePath string, config *Config) error {
	info, err := os.Stat(filePath)
	if err != nil {
		config.debugInfo("File does not exist, skipping")
		return nil
	}
	if info.Size() == 0 {
		config.debugInfo("File is empty, skipping")
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Seek(-1, io.SeekEnd)
	if err != nil {
		return err
	}

	var lastByte [1]byte
	_, err = file.Read(lastByte[:])
	if err != nil {
		return err
	}

	if lastByte[0] != newlineByte {
		config.debugInfo("Adding newline (missing)")
		config.debugFileContents(filePath)

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

	config.debugInfo("Already ends with newline")
	config.debugFileContents(filePath)
	return nil
}

func AddNewlineIfNeeded(filePath string, debugMode bool) error {
	config := &Config{Debug: debugMode, Silent: false}
	return addNewlineIfNeeded(filePath, config)
}
