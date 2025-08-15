package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// MockLogger implements Logger interface for testing
type MockLogger struct {
	Messages      []string
	DebugMessages []string
	Sections      []string
	Separators    int
}

// Log records regular messages
func (m *MockLogger) Log(format string, args ...any) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, args...))
}

// Debug records debug messages
func (m *MockLogger) Debug(format string, args ...any) {
	m.DebugMessages = append(m.DebugMessages, fmt.Sprintf(format, args...))
}

// DebugSection records section starts
func (m *MockLogger) DebugSection(title string) {
	m.Sections = append(m.Sections, title)
}

// DebugSeparator counts separators
func (m *MockLogger) DebugSeparator() {
	m.Separators++
}

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()
	w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestNeedsNewlineFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "content with newline",
			content:  []byte("hello\n"),
			expected: false,
		},
		{
			name:     "content without newline",
			content:  []byte("hello"),
			expected: true,
		},
		{
			name:     "content with only newline",
			content:  []byte("\n"),
			expected: false,
		},
		{
			name:     "multiline with newline",
			content:  []byte("line1\nline2\n"),
			expected: false,
		},
		{
			name:     "multiline without newline",
			content:  []byte("line1\nline2"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsNewlineFromContent(tt.content)
			if result != tt.expected {
				t.Errorf("needsNewlineFromContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")
	_ = os.WriteFile(existingFile, []byte("content"), 0o644)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "existing file",
			filePath: existingFile,
			expected: true,
		},
		{
			name:     "non-existent file",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
		},
		{
			name:     "directory",
			filePath: tempDir,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.filePath)
			if result != tt.expected {
				t.Errorf("fileExists() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsFileEmpty(t *testing.T) {
	tempDir := t.TempDir()
	emptyFile := filepath.Join(tempDir, "empty.txt")
	nonEmptyFile := filepath.Join(tempDir, "nonempty.txt")
	_ = os.WriteFile(emptyFile, []byte{}, 0o644)
	_ = os.WriteFile(nonEmptyFile, []byte("content"), 0o644)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "empty file",
			filePath: emptyFile,
			expected: true,
		},
		{
			name:     "non-empty file",
			filePath: nonEmptyFile,
			expected: false,
		},
		{
			name:     "non-existent file",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFileEmpty(tt.filePath)
			if result != tt.expected {
				t.Errorf("isFileEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		jsonText string
		expected []string
	}{
		{
			name:     "invalid json",
			jsonText: "not json",
			expected: nil,
		},
		{
			name:     "no tool_input",
			jsonText: `{"other": "data"}`,
			expected: nil,
		},
		{
			name:     "path field",
			jsonText: `{"tool_input": {"path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "file_path field",
			jsonText: `{"tool_input": {"file_path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "paths array",
			jsonText: `{"tool_input": {"paths": ["/test/file1.txt", "/test/file2.txt"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.txt"},
		},
		{
			name:     "multiple fields",
			jsonText: `{"tool_input": {"path": "/test/file1.txt", "file_path": "/test/file2.txt", "paths": ["/test/file3.txt"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.txt", "/test/file3.txt"},
		},
		{
			name:     "empty path",
			jsonText: `{"tool_input": {"path": ""}}`,
			expected: nil,
		},
		{
			name:     "paths with empty strings",
			jsonText: `{"tool_input": {"paths": ["", "/test/file.txt", ""]}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "non-string path field",
			jsonText: `{"tool_input": {"path": 123}}`,
			expected: nil,
		},
		{
			name:     "non-string file_path field",
			jsonText: `{"tool_input": {"file_path": true}}`,
			expected: nil,
		},
		{
			name:     "non-array paths field",
			jsonText: `{"tool_input": {"paths": "not-an-array"}}`,
			expected: nil,
		},
		{
			name:     "paths array with non-string elements",
			jsonText: `{"tool_input": {"paths": ["/valid/file.txt", 123, null, "/another/file.txt"]}}`,
			expected: []string{"/valid/file.txt", "/another/file.txt"},
		},
		{
			name:     "nested tool_input is not object",
			jsonText: `{"tool_input": "not-an-object"}`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilePaths(tt.jsonText)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractFilePaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAddNewlineIfNeededBasicCases(t *testing.T) {
	tests := []struct {
		name            string
		initialContent  string
		expectedContent string
	}{
		{
			name:            "file without newline gets newline added",
			initialContent:  "content",
			expectedContent: "content\n",
		},
		{
			name:            "file with newline remains unchanged",
			initialContent:  "content\n",
			expectedContent: "content\n",
		},
		{
			name:            "empty file remains unchanged",
			initialContent:  "",
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, []byte(tt.initialContent), 0o644)

			logger := NewConsoleLogger(&Config{})
			err := addNewlineIfNeeded(testFile, logger)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			content, _ := os.ReadFile(testFile)
			if string(content) != tt.expectedContent {
				t.Errorf("Expected %q, got %q", tt.expectedContent, string(content))
			}
		})
	}
}

func TestAddNewlineIfNeededErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		fileType    string
		expectError bool
	}{
		{
			name:        "non-existent file returns no error",
			fileType:    "nonexistent",
			expectError: false,
		},
		{
			name:        "directory returns error",
			fileType:    "directory",
			expectError: true,
		},
		{
			name:        "read-only file returns error",
			fileType:    "readonly",
			expectError: true,
		},
	}

	tempDir := t.TempDir()
	logger := NewConsoleLogger(&Config{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			var cleanup func()

			switch tt.fileType {
			case "nonexistent":
				filePath = "/nonexistent/file.txt"
			case "directory":
				filePath = tempDir
			case "readonly":
				filePath = filepath.Join(tempDir, "readonly.txt")
				_ = os.WriteFile(filePath, []byte("content"), 0o644)
				_ = os.Chmod(filePath, 0o444)
				cleanup = func() { _ = os.Chmod(filePath, 0o644) }
			}

			if cleanup != nil {
				defer cleanup()
			}

			err := addNewlineIfNeeded(filePath, logger)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConsoleLoggerOutputModes(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		logFunc        func(Logger)
		expectedOutput string
		shouldContain  bool
	}{
		{
			name:           "normal mode outputs message",
			config:         &Config{Debug: false, Silent: false},
			logFunc:        func(l Logger) { l.Log("test message") },
			expectedOutput: "test message",
			shouldContain:  false,
		},
		{
			name:           "silent mode outputs nothing",
			config:         &Config{Debug: false, Silent: true},
			logFunc:        func(l Logger) { l.Log("test message") },
			expectedOutput: "",
			shouldContain:  false,
		},
		{
			name:           "debug mode outputs debug info",
			config:         &Config{Debug: true, Silent: false},
			logFunc:        func(l Logger) { l.Debug("debug message") },
			expectedOutput: "debug message",
			shouldContain:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewConsoleLogger(tt.config)
			output := captureOutput(func() {
				tt.logFunc(logger)
			})

			if tt.shouldContain {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got: %q", tt.expectedOutput, output)
				}
			} else {
				if output != tt.expectedOutput {
					t.Errorf("Expected %q, got: %q", tt.expectedOutput, output)
				}
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *Config
	}{
		{
			name:     "no flags",
			args:     []string{"ccnewline"},
			expected: &Config{Debug: false, Silent: false},
		},
		{
			name:     "debug flag -d",
			args:     []string{"ccnewline", "-d"},
			expected: &Config{Debug: true, Silent: false},
		},
		{
			name:     "debug flag --debug",
			args:     []string{"ccnewline", "--debug"},
			expected: &Config{Debug: true, Silent: false},
		},
		{
			name:     "silent flag -s",
			args:     []string{"ccnewline", "-s"},
			expected: &Config{Debug: false, Silent: true},
		},
		{
			name:     "silent flag --silent",
			args:     []string{"ccnewline", "--silent"},
			expected: &Config{Debug: false, Silent: true},
		},
		{
			name:     "both flags",
			args:     []string{"ccnewline", "-d", "-s"},
			expected: &Config{Debug: true, Silent: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			config := parseFlags()

			if !reflect.DeepEqual(config, tt.expected) {
				t.Errorf("parseFlags() = %+v, want %+v", config, tt.expected)
			}
		})
	}
}

func TestParseFilePathsFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input with multiple path fields",
			input:    `{"tool_input": {"path": "/path1", "file_path": "/path2", "paths": ["/path3", "/path4"]}}`,
			expected: []string{"/path1", "/path2", "/path3", "/path4"},
		},
		{
			name:     "JSON input with file_path only",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: []string{"/test.txt"},
		},
		{
			name:     "plain text input",
			input:    "/file1.txt\n/file2.txt\n\n/file3.txt",
			expected: []string{"/file1.txt", "/file2.txt", "/file3.txt"},
		},
		{
			name:     "plain text with empty lines",
			input:    "/file1.txt\n\n/file2.txt\n\n",
			expected: []string{"/file1.txt", "/file2.txt"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   \n  \n",
			expected: nil,
		},
		{
			name:     "single file path",
			input:    "/single/file.txt",
			expected: []string{"/single/file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFilePathsFromText(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseFilePathsFromText() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReadFilePathsFromReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input with file_path",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: []string{"/test.txt"},
		},
		{
			name:     "plain text input",
			input:    "/file1.txt\n/file2.txt",
			expected: []string{"/file1.txt", "/file2.txt"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			mockLogger := &MockLogger{}

			result := readFilePathsFromReader(mockLogger, reader)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("readFilePathsFromReader() = %v, want %v", result, tt.expected)
			}

			// Verify logger was called for non-empty input
			if tt.expected != nil && len(mockLogger.Sections) == 0 {
				t.Error("Expected debug sections to be logged")
			}
		})
	}
}

func TestConsoleLogger(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		message        string
		action         func(Logger, string)
		expectedOutput string
	}{
		{
			name:           "Log in silent mode",
			config:         &Config{Silent: true},
			message:        "test message\n",
			action:         func(l Logger, msg string) { l.Log(msg) },
			expectedOutput: "",
		},
		{
			name:           "Log in debug mode",
			config:         &Config{Debug: true},
			message:        "test message\n",
			action:         func(l Logger, msg string) { l.Log(msg) },
			expectedOutput: "",
		},
		{
			name:           "Log in normal mode",
			config:         &Config{},
			message:        "test message\n",
			action:         func(l Logger, msg string) { l.Log(msg) },
			expectedOutput: "test message\n",
		},
		{
			name:           "Debug without debug mode",
			config:         &Config{Debug: false},
			message:        "test",
			action:         func(l Logger, msg string) { l.Debug(msg) },
			expectedOutput: "",
		},
		{
			name:           "Debug with debug mode",
			config:         &Config{Debug: true},
			message:        "debug message",
			action:         func(l Logger, msg string) { l.Debug(msg) },
			expectedOutput: "debug message",
		},
		{
			name:           "DebugSection with debug mode",
			config:         &Config{Debug: true},
			message:        "TEST",
			action:         func(l Logger, msg string) { l.DebugSection(msg) },
			expectedOutput: "TEST",
		},
		{
			name:           "DebugSeparator with debug mode",
			config:         &Config{Debug: true},
			message:        "",
			action:         func(l Logger, msg string) { l.DebugSeparator() },
			expectedOutput: "└",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewConsoleLogger(tt.config)
			output := captureOutput(func() {
				tt.action(logger, tt.message)
			})

			if tt.expectedOutput == "" {
				if output != "" {
					t.Errorf("Expected no output, got: %q", output)
				}
			} else {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got: %q", tt.expectedOutput, output)
				}
			}
		})
	}
}
