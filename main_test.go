package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

func TestAddNewlineIfNeeded(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		expectError bool
		expectAdd   bool
		debugMode   bool
	}{
		{
			name:        "file with newline",
			fileContent: "content\n",
			expectError: false,
			expectAdd:   false,
			debugMode:   false,
		},
		{
			name:        "file without newline",
			fileContent: "content",
			expectError: false,
			expectAdd:   true,
			debugMode:   true,
		},
		{
			name:        "empty file",
			fileContent: "",
			expectError: false,
			expectAdd:   false,
			debugMode:   false,
		},
		{
			name:        "only newline",
			fileContent: "\n",
			expectError: false,
			expectAdd:   false,
			debugMode:   true,
		},
		{
			name:        "multiline content without newline",
			fileContent: "line1\nline2\nline3",
			expectError: false,
			expectAdd:   true,
			debugMode:   true,
		},
		{
			name:        "multiline content with newline",
			fileContent: "line1\nline2\nline3\n",
			expectError: false,
			expectAdd:   false,
			debugMode:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".txt")

			if tt.fileContent != "" || tt.name == "empty file" {
				err := os.WriteFile(testFile, []byte(tt.fileContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			config := &Config{Debug: tt.debugMode, Silent: false}
			logger := NewConsoleLogger(config)
			err := addNewlineIfNeeded(testFile, logger)

			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && (tt.fileContent != "" || tt.name == "empty file") {
				content, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("Failed to read test file: %v", err)
				}

				if tt.expectAdd {
					expectedContent := tt.fileContent + "\n"
					if string(content) != expectedContent {
						t.Errorf("Expected content %q, got %q", expectedContent, string(content))
					}
				} else if string(content) != tt.fileContent {
					t.Errorf("Expected content %q, got %q", tt.fileContent, string(content))
				}
			}
		})
	}

	// Error handling scenarios
	t.Run("non-existent file", func(t *testing.T) {
		config := &Config{Debug: false, Silent: false}
		logger := NewConsoleLogger(config)
		err := addNewlineIfNeeded("/nonexistent/file.txt", logger)
		if err != nil {
			t.Errorf("Expected no error for non-existent file, got: %v", err)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		config := &Config{Debug: false, Silent: false}
		logger := NewConsoleLogger(config)
		err := addNewlineIfNeeded(tempDir, logger)
		if err == nil {
			t.Error("Expected error when processing directory, got nil")
		}
	})

	t.Run("read-only file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "readonly.txt")

		err := os.WriteFile(testFile, []byte("content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = os.Chmod(testFile, 0o444)
		if err != nil {
			t.Fatalf("Failed to make file read-only: %v", err)
		}

		config := &Config{Debug: false, Silent: false}
		logger := NewConsoleLogger(config)
		err = addNewlineIfNeeded(testFile, logger)
		if err == nil {
			t.Error("Expected error when trying to append to read-only file, got nil")
		}

		_ = os.Chmod(testFile, 0o644) // Restore for cleanup
	})
}

func TestReadFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "valid JSON with file_path",
			input:    `{"tool_input": {"file_path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "valid JSON with paths array",
			input:    `{"tool_input": {"paths": ["/test/file1.txt", "/test/file2.txt"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.txt"},
		},
		{
			name:     "invalid JSON fallback to plain text",
			input:    "/test/file1.txt\n/test/file2.txt\n",
			expected: []string{"/test/file1.txt", "/test/file2.txt"},
		},
		{
			name:     "plain text with empty lines",
			input:    "/test/file1.txt\n\n/test/file2.txt\n\n",
			expected: []string{"/test/file1.txt", "/test/file2.txt"},
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
			name:     "multiline input triggering debug truncation",
			input:    "line1\nline2\nline3\nline4\nline5",
			expected: []string{"line1", "line2", "line3", "line4", "line5"},
		},
		{
			name:     "no stdin available",
			input:    "", // Special case - will be handled differently
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			if tt.name == "no stdin available" {
				// Special case: simulate no stdin available
				devNull, err := os.Open("/dev/null")
				if err != nil {
					t.Fatalf("Failed to open /dev/null: %v", err)
				}
				defer devNull.Close()
				os.Stdin = devNull
			} else {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatalf("Failed to create pipe: %v", err)
				}
				defer r.Close()

				os.Stdin = r

				go func() {
					defer w.Close()
					_, _ = w.Write([]byte(tt.input))
				}()
			}

			config := &Config{Debug: false}
			logger := NewConsoleLogger(config)
			result := readFilePaths(logger)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("readFilePaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOutputModes(t *testing.T) {
	tests := []struct {
		name         string
		debug        bool
		silent       bool
		expectOutput bool
		expectedText string
	}{
		{
			name:         "normal mode - should output message",
			debug:        false,
			silent:       false,
			expectOutput: true,
			expectedText: "Added newline to",
		},
		{
			name:         "silent mode - should not output",
			debug:        false,
			silent:       true,
			expectOutput: false,
			expectedText: "",
		},
		{
			name:         "debug mode - should output debug info",
			debug:        true,
			silent:       false,
			expectOutput: true,
			expectedText: "Adding newline (missing)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")

			// Create test file without newline
			err := os.WriteFile(testFile, []byte("test content"), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()
			os.Stdout = w

			// Test the function directly
			config := &Config{Debug: tt.debug, Silent: tt.silent}
			logger := NewConsoleLogger(config)
			err = addNewlineIfNeeded(testFile, logger)
			if err != nil {
				t.Fatalf("addNewlineIfNeeded failed: %v", err)
			}

			w.Close()

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if tt.expectOutput {
				if output == "" {
					t.Errorf("Expected output but got none")
				}
				if tt.expectedText != "" && !strings.Contains(output, tt.expectedText) {
					t.Errorf("Expected output to contain '%s', got: %s", tt.expectedText, output)
				}
			} else if output != "" {
				t.Errorf("Expected no output but got: %s", output)
			}

			// Verify file was modified
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}
			if !strings.HasSuffix(string(content), "\n") {
				t.Error("Expected test file to end with newline")
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

func TestReadFilePathsFromReaderWithDebugOutput(t *testing.T) {
	input := `{"tool_input": {"file_path": "/test.txt"}}`
	reader := strings.NewReader(input)
	logger := NewConsoleLogger(&Config{Debug: true})

	output := captureOutput(func() {
		readFilePathsFromReader(logger, reader)
	})

	expectedStrings := []string{"INPUT PARSING", "JSON parsing successful"}
	for _, expectedStr := range expectedStrings {
		if !strings.Contains(output, expectedStr) {
			t.Errorf("Expected debug output to contain '%s'", expectedStr)
		}
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

func TestMockLogger(t *testing.T) {
	tests := []struct {
		name             string
		actions          func(*MockLogger)
		expectedMessages int
		expectedDebug    int
		expectedSections int
		expectedSeps     int
	}{
		{
			name: "captures all calls",
			actions: func(m *MockLogger) {
				m.Log("test log")
				m.Debug("test debug")
				m.DebugSection("TEST")
				m.DebugSeparator()
			},
			expectedMessages: 1,
			expectedDebug:    1,
			expectedSections: 1,
			expectedSeps:     1,
		},
		{
			name: "multiple calls",
			actions: func(m *MockLogger) {
				m.Log("msg1")
				m.Log("msg2")
				m.Debug("debug1")
				m.DebugSeparator()
				m.DebugSeparator()
			},
			expectedMessages: 2,
			expectedDebug:    1,
			expectedSections: 0,
			expectedSeps:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockLogger{}
			tt.actions(mock)

			if len(mock.Messages) != tt.expectedMessages {
				t.Errorf("Expected %d messages, got %d", tt.expectedMessages, len(mock.Messages))
			}
			if len(mock.DebugMessages) != tt.expectedDebug {
				t.Errorf("Expected %d debug messages, got %d", tt.expectedDebug, len(mock.DebugMessages))
			}
			if len(mock.Sections) != tt.expectedSections {
				t.Errorf("Expected %d sections, got %d", tt.expectedSections, len(mock.Sections))
			}
			if mock.Separators != tt.expectedSeps {
				t.Errorf("Expected %d separators, got %d", tt.expectedSeps, mock.Separators)
			}
		})
	}
}

func TestFunctionsWithMockLogger(t *testing.T) {
	tests := []struct {
		name                 string
		action               func(*MockLogger) (any, error)
		expectedResult       any
		expectedError        bool
		expectedSections     []string
		expectedDebugContent []string
		expectedSeparators   int
	}{
		{
			name: "processFiles logs correct debug info",
			action: func(mock *MockLogger) (any, error) {
				filePaths := []string{"/file1.txt", "/file2.txt"}
				processFiles(mock, filePaths)
				return nil, nil
			},
			expectedSections:   []string{"PROCESSING"},
			expectedSeparators: 1,
		},
		{
			name: "addNewlineIfNeeded with non-existent file",
			action: func(mock *MockLogger) (any, error) {
				err := addNewlineIfNeeded("/nonexistent/file.txt", mock)
				return nil, err
			},
			expectedError:        false,
			expectedDebugContent: []string{"does not exist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockLogger{}
			result, err := tt.action(mock)

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Error("Expected error, but got none")
			} else if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Check result if specified
			if tt.expectedResult != nil && !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}

			// Check sections
			for _, expectedSection := range tt.expectedSections {
				if !slices.Contains(mock.Sections, expectedSection) {
					t.Errorf("Expected section '%s' to be logged", expectedSection)
				}
			}

			// Check debug content
			for _, expectedContent := range tt.expectedDebugContent {
				found := false
				for _, msg := range mock.DebugMessages {
					if strings.Contains(msg, expectedContent) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected debug message containing '%s'", expectedContent)
				}
			}

			// Check separators
			if tt.expectedSeparators > 0 && mock.Separators < tt.expectedSeparators {
				t.Errorf("Expected at least %d separators, got %d", tt.expectedSeparators, mock.Separators)
			}
		})
	}

	// Special test for shouldProcessFile with empty file
	t.Run("shouldProcessFile with empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		emptyFile := filepath.Join(tempDir, "empty.txt")
		if err := os.WriteFile(emptyFile, []byte{}, 0o644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		mock := &MockLogger{}
		result := shouldProcessFile(emptyFile, mock)

		if result {
			t.Error("Expected false for empty file")
		}

		// Check debug message
		found := false
		for _, msg := range mock.DebugMessages {
			if strings.Contains(msg, "empty") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'empty' debug message")
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("no files to process", func(t *testing.T) {
		config := &Config{Debug: false, Silent: false}
		reader := strings.NewReader("")
		run(config, reader) // Should not panic or error
	})

	t.Run("integration test - JSON input with file processing", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte("content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		input := `{"tool_input": {"file_path": "` + testFile + `"}}`
		config := &Config{Debug: false, Silent: false}

		// Capture stdout
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer r.Close()
		os.Stdout = w

		reader := strings.NewReader(input)
		run(config, reader)
		w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Added newline to") {
			t.Errorf("Expected output to contain 'Added newline to', got: %s", output)
		}
	})
}
