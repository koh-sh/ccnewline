package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

			err := AddNewlineIfNeeded(testFile, tt.debugMode)

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
		err := AddNewlineIfNeeded("/nonexistent/file.txt", false)
		if err != nil {
			t.Errorf("Expected no error for non-existent file, got: %v", err)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		err := AddNewlineIfNeeded(tempDir, false)
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

		err = AddNewlineIfNeeded(testFile, false)
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
			result := readFilePaths(config)

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
			err = addNewlineIfNeeded(testFile, config)
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
