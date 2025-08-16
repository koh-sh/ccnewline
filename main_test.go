package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koh-sh/ccnewline/internal/cli"
	"github.com/koh-sh/ccnewline/internal/logging"
	"github.com/koh-sh/ccnewline/internal/processing"
	"github.com/koh-sh/ccnewline/internal/toolinput"
)

// Integration test for the main entry point
func TestMainEntryPoint(t *testing.T) {
	// Test that main function exists and can be called
	// This is primarily a compilation test
	config := &cli.Config{
		Debug:  false,
		Silent: true,
	}

	logger := logging.NewConsoleLogger(config)
	if logger == nil {
		t.Error("NewConsoleLogger should return a valid logger")
	}
}

// Integration test for core functionality
func TestCoreWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file without newline
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Test JSON input parsing
	jsonInput := `{"tool_input": {"path": "` + testFile + `"}}`
	paths, err := toolinput.ParseToolInput(jsonInput)
	if err != nil {
		t.Fatal(err)
	}

	if len(paths) != 1 || paths[0] != testFile {
		t.Errorf("Expected [%s], got %v", testFile, paths)
	}

	// Test file processing
	config := &cli.Config{Silent: true}
	logger := logging.NewConsoleLogger(config)

	input := strings.NewReader(jsonInput)
	processing.Run(config, logger, input)

	// Verify file now has newline
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(string(content), "\n") {
		t.Error("File should now end with newline")
	}
}
