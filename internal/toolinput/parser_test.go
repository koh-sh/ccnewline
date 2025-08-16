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
