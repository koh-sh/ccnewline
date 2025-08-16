package toolinput

import (
	"strings"
	"testing"
)

func TestParseToolInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: []string{"not json"},
		},
		{
			name:     "JSON without tool_input",
			input:    `{"other": "data"}`,
			expected: []string{},
		},
		{
			name:     "Edit tool with path field",
			input:    `{"tool_input": {"path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "Write tool with file_path field",
			input:    `{"tool_input": {"file_path": "/test/file.go"}}`,
			expected: []string{"/test/file.go"},
		},
		{
			name:     "MultiEdit tool with paths array",
			input:    `{"tool_input": {"paths": ["/test/file1.txt", "/test/file2.go"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "Multiple fields",
			input:    `{"tool_input": {"path": "/test/file1.txt", "file_path": "/test/file2.go"}}`,
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseToolInput(tt.input)
			if err != nil {
				t.Errorf("ParseToolInput() error = %v", err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ParseToolInput() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("ParseToolInput()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestPathExtractorParseJSON(t *testing.T) {
	extractor := newPathExtractor()

	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name:     "valid JSON with path",
			input:    `{"tool_input": {"path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
			hasError: false,
		},
		{
			name:     "valid JSON with file_path",
			input:    `{"tool_input": {"file_path": "/test/file.go"}}`,
			expected: []string{"/test/file.go"},
			hasError: false,
		},
		{
			name:     "valid JSON with paths array",
			input:    `{"tool_input": {"paths": ["/test/file1.txt", "/test/file2.go"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.go"},
			hasError: false,
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.parseJSON(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("parseJSON() error = %v, hasError %v", err, tt.hasError)
				return
			}
			if !tt.hasError {
				if len(result) != len(tt.expected) {
					t.Errorf("parseJSON() length = %v, want %v", len(result), len(tt.expected))
					return
				}
				for i, path := range result {
					if path != tt.expected[i] {
						t.Errorf("parseJSON()[%d] = %v, want %v", i, path, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestPathExtractorParsePlainText(t *testing.T) {
	extractor := newPathExtractor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "plain text input",
			input:    "/test/file1.txt\n/test/file2.go\n",
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "plain text with empty lines",
			input:    "/test/file1.txt\n\n/test/file2.go\n\n",
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "single line",
			input:    "/test/file.txt",
			expected: []string{"/test/file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.parsePlainText(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parsePlainText() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("parsePlainText()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestReadInputLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "normal input",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "input with trailing empty lines",
			input:    "line1\nline2\n\n\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "input with leading empty lines",
			input:    "\n\nline1\nline2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "input with both leading and trailing empty lines",
			input:    "\n\nline1\nline2\n\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result := readInputLines(reader)
			if len(result) != len(tt.expected) {
				t.Errorf("readInputLines() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("readInputLines()[%d] = %v, want %v", i, line, tt.expected[i])
				}
			}
		})
	}
}

// Mock logger for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(message string) {
	m.debugMessages = append(m.debugMessages, message)
}

func (m *mockLogger) Info(message string) {
	m.infoMessages = append(m.infoMessages, message)
}

func (m *mockLogger) Error(message string) {
	m.errorMessages = append(m.errorMessages, message)
}

func (m *mockLogger) LogFileProcessing(file, result string) {
	m.infoMessages = append(m.infoMessages, file+": "+result)
}

func (m *mockLogger) ShowProcessingStart(files []string) {
	m.debugMessages = append(m.debugMessages, "Processing start")
}

func (m *mockLogger) ShowProcessingEnd(totalFiles, processedFiles int) {
	m.debugMessages = append(m.debugMessages, "Processing end")
}

func TestReadToolInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input with Edit tool",
			input:    `{"tool_input": {"file_path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "JSON input with MultiEdit tool",
			input:    `{"tool_input": {"paths": ["/test/file1.txt", "/test/file2.go"]}}`,
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "Plain text input",
			input:    "/test/file1.txt\n/test/file2.go",
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Invalid JSON",
			input:    "not valid json",
			expected: []string{"not valid json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			reader := strings.NewReader(tt.input)

			result := ReadToolInput(logger, reader)

			if len(result) != len(tt.expected) {
				t.Errorf("ReadToolInput() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("ReadToolInput()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestPathExtractorIsJSON(t *testing.T) {
	extractor := newPathExtractor()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid JSON object",
			input:    `{"tool_input": {"path": "/test/file.txt"}}`,
			expected: true,
		},
		{
			name:     "valid JSON array",
			input:    `[1, 2, 3]`,
			expected: false, // isJSON only checks for objects, not arrays
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "malformed JSON",
			input:    `{"tool_input": {"path": "/test/file.txt"}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.isJSON(tt.input)
			if result != tt.expected {
				t.Errorf("isJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPathExtractorExtractPathsFromToolInput(t *testing.T) {
	extractor := newPathExtractor()

	tests := []struct {
		name     string
		toolData map[string]any
		expected []string
	}{
		{
			name: "path field only",
			toolData: map[string]any{
				"path": "/test/file.txt",
			},
			expected: []string{"/test/file.txt"},
		},
		{
			name: "file_path field only",
			toolData: map[string]any{
				"file_path": "/test/file.go",
			},
			expected: []string{"/test/file.go"},
		},
		{
			name: "paths array",
			toolData: map[string]any{
				"paths": []any{"/test/file1.txt", "/test/file2.go"},
			},
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name: "multiple fields",
			toolData: map[string]any{
				"path":      "/test/file1.txt",
				"file_path": "/test/file2.go",
				"paths":     []any{"/test/file3.md"},
			},
			expected: []string{"/test/file1.txt", "/test/file2.go", "/test/file3.md"},
		},
		{
			name: "no relevant fields",
			toolData: map[string]any{
				"other": "data",
			},
			expected: []string{},
		},
		{
			name: "invalid paths array",
			toolData: map[string]any{
				"paths": "not an array",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.extractPathsFromToolInput(tt.toolData)
			if len(result) != len(tt.expected) {
				t.Errorf("extractPathsFromToolInput() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("extractPathsFromToolInput()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestInputCheckerCheckAvailability(t *testing.T) {
	// This function primarily checks stdin availability
	// For non-stdin readers, it always returns true
	checker := newInputChecker()
	logger := &mockLogger{}
	reader := strings.NewReader("some content")

	result := checker.checkAvailability(logger, reader)
	if !result {
		t.Error("checkAvailability() should return true for non-stdin readers")
	}
}

func TestInputReaderReadPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input",
			input:    `{"tool_input": {"file_path": "/test/file.txt"}}`,
			expected: []string{"/test/file.txt"},
		},
		{
			name:     "plain text input",
			input:    "/test/file1.txt\n/test/file2.go",
			expected: []string{"/test/file1.txt", "/test/file2.go"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only input",
			input:    "   \n\t  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			reader := newInputReader()
			inputReader := strings.NewReader(tt.input)

			result := reader.readPaths(logger, inputReader)

			if len(result) != len(tt.expected) {
				t.Errorf("readPaths() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("readPaths()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestParseToolInputEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name:     "nested tool_input",
			input:    `{"nested": {"tool_input": {"path": "/test/file.txt"}}}`,
			expected: []string{},
			hasError: false,
		},
		{
			name:     "tool_input as array",
			input:    `{"tool_input": ["/test/file.txt"]}`,
			expected: []string{},
			hasError: false,
		},
		{
			name:     "tool_input as string",
			input:    `{"tool_input": "/test/file.txt"}`,
			expected: []string{},
			hasError: false,
		},
		{
			name:     "mixed types in paths array",
			input:    `{"tool_input": {"paths": ["/test/file.txt", 123, true]}}`,
			expected: []string{"/test/file.txt"},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseToolInput(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("ParseToolInput() error = %v, hasError %v", err, tt.hasError)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ParseToolInput() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("ParseToolInput()[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}
		})
	}
}
