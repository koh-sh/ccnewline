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
	SectionEnds   int
}

// log records regular messages
func (m *MockLogger) log(format string, args ...any) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, args...))
}

// debug records debug messages
func (m *MockLogger) debug(format string, args ...any) {
	m.DebugMessages = append(m.DebugMessages, fmt.Sprintf(format, args...))
}

// debugSection records section starts
func (m *MockLogger) debugSection(title string) {
	m.Sections = append(m.Sections, title)
}

// debugEnd counts section ends
func (m *MockLogger) debugEnd() {
	m.SectionEnds++
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

			logger := newConsoleLogger(&config{})
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
	tempDir := t.TempDir()
	logger := newConsoleLogger(&config{})

	// Setup readonly file
	readonlyFile := filepath.Join(tempDir, "readonly.txt")
	_ = os.WriteFile(readonlyFile, []byte("content"), 0o644)
	_ = os.Chmod(readonlyFile, 0o444)
	defer func() { _ = os.Chmod(readonlyFile, 0o644) }() // cleanup

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "non-existent file returns no error",
			filePath:    "/nonexistent/file.txt",
			expectError: false,
		},
		{
			name:        "directory returns error",
			filePath:    tempDir,
			expectError: true,
		},
		{
			name:        "read-only file returns error",
			filePath:    readonlyFile,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := addNewlineIfNeeded(tt.filePath, logger)

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
		config         *config
		methodType     string
		message        string
		expectedOutput string
		containsCheck  bool
	}{
		{
			name:           "normal mode outputs message",
			config:         &config{Debug: false, Silent: false},
			methodType:     "Log",
			message:        "test message",
			expectedOutput: "test message",
			containsCheck:  false,
		},
		{
			name:           "silent mode outputs nothing",
			config:         &config{Debug: false, Silent: true},
			methodType:     "Log",
			message:        "test message",
			expectedOutput: "",
			containsCheck:  false,
		},
		{
			name:           "debug mode outputs debug info",
			config:         &config{Debug: true, Silent: false},
			methodType:     "Debug",
			message:        "debug message",
			expectedOutput: "debug message",
			containsCheck:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newConsoleLogger(tt.config)
			output := captureOutput(func() {
				switch tt.methodType {
				case "Log":
					logger.log(tt.message)
				case "Debug":
					logger.debug(tt.message)
				}
			})

			if tt.containsCheck {
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
		expected *config
	}{
		{
			name:     "no flags",
			args:     []string{"ccnewline"},
			expected: &config{Debug: false, Silent: false},
		},
		{
			name:     "debug flag -d",
			args:     []string{"ccnewline", "-d"},
			expected: &config{Debug: true, Silent: false},
		},
		{
			name:     "debug flag --debug",
			args:     []string{"ccnewline", "--debug"},
			expected: &config{Debug: true, Silent: false},
		},
		{
			name:     "silent flag -s",
			args:     []string{"ccnewline", "-s"},
			expected: &config{Debug: false, Silent: true},
		},
		{
			name:     "silent flag --silent",
			args:     []string{"ccnewline", "--silent"},
			expected: &config{Debug: false, Silent: true},
		},
		{
			name:     "both flags",
			args:     []string{"ccnewline", "-d", "-s"},
			expected: &config{Debug: true, Silent: true},
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
		config         *config
		message        string
		methodType     string
		expectedOutput string
		containsCheck  bool
	}{
		{
			name:           "Log in silent mode",
			config:         &config{Silent: true},
			message:        "test message\n",
			methodType:     "Log",
			expectedOutput: "",
			containsCheck:  false,
		},
		{
			name:           "Log in debug mode",
			config:         &config{Debug: true},
			message:        "test message\n",
			methodType:     "Log",
			expectedOutput: "",
			containsCheck:  false,
		},
		{
			name:           "Log in normal mode",
			config:         &config{},
			message:        "test message\n",
			methodType:     "Log",
			expectedOutput: "test message\n",
			containsCheck:  false,
		},
		{
			name:           "Debug without debug mode",
			config:         &config{Debug: false},
			message:        "test",
			methodType:     "Debug",
			expectedOutput: "",
			containsCheck:  false,
		},
		{
			name:           "Debug with debug mode",
			config:         &config{Debug: true},
			message:        "debug message",
			methodType:     "Debug",
			expectedOutput: "debug message",
			containsCheck:  true,
		},
		{
			name:           "DebugSection with debug mode",
			config:         &config{Debug: true},
			message:        "TEST",
			methodType:     "DebugSection",
			expectedOutput: "TEST",
			containsCheck:  true,
		},
		{
			name:           "DebugSeparator with debug mode",
			config:         &config{Debug: true},
			message:        "",
			methodType:     "DebugSeparator",
			expectedOutput: "└",
			containsCheck:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newConsoleLogger(tt.config)
			output := captureOutput(func() {
				switch tt.methodType {
				case "Log":
					logger.log(tt.message)
				case "Debug":
					logger.debug(tt.message)
				case "DebugSection":
					logger.debugSection(tt.message)
				case "DebugSeparator":
					logger.debugEnd()
				}
			})

			switch {
			case tt.expectedOutput == "":
				if output != "" {
					t.Errorf("Expected no output, got: %q", output)
				}
			case tt.containsCheck:
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got: %q", tt.expectedOutput, output)
				}
			default:
				if output != tt.expectedOutput {
					t.Errorf("Expected %q, got: %q", tt.expectedOutput, output)
				}
			}
		})
	}
}

// TestVersionHandler tests version handling functionality
func TestVersionHandler(t *testing.T) {
	tests := []struct {
		name        string
		checkType   string
		expectError bool
	}{
		{
			name:        "versionHandler creation",
			checkType:   "creation",
			expectError: false,
		},
		{
			name:        "FlagParser has versionHandler",
			checkType:   "flagparser",
			expectError: false,
		},
		{
			name:        "versionHandler is not nil",
			checkType:   "notnull",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch tt.checkType {
			case "creation":
				vh := &versionHandler{}
				// versionHandler creation always succeeds
				_ = vh // Just verify it can be created
			case "flagparser":
				fp := newFlagParser()
				if fp.versionHandler == nil {
					err = fmt.Errorf("FlagParser.versionHandler is nil")
				}
			case "notnull":
				if versionHandler := newFlagParser().versionHandler; versionHandler == nil {
					err = fmt.Errorf("versionHandler should not be nil")
				}
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Note: ShowVersion() calls os.Exit(0), so we can't test it directly
			// in a unit test. Integration tests would be more appropriate.
		})
	}
}

// TestArgumentValidator tests argument validation
func TestArgumentValidator(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no arguments",
			args:        []string{"ccnewline"},
			expectError: false,
		},
		{
			name:        "with arguments",
			args:        []string{"ccnewline", "arg1", "arg2"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args
			flag.Parse()

			av := &argumentValidator{}
			err := av.validateArgs()

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestFlagParser tests the flag parsing functionality
func TestFlagParser(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *config
	}{
		{
			name:     "no flags",
			args:     []string{"ccnewline"},
			expected: &config{Debug: false, Silent: false},
		},
		{
			name:     "debug flag",
			args:     []string{"ccnewline", "-d"},
			expected: &config{Debug: true, Silent: false},
		},
		{
			name:     "silent flag",
			args:     []string{"ccnewline", "-s"},
			expected: &config{Debug: false, Silent: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			fp := newFlagParser()
			config := fp.parse()

			if !reflect.DeepEqual(config, tt.expected) {
				t.Errorf("FlagParser.Parse() = %+v, want %+v", config, tt.expected)
			}
		})
	}
}

// TestInputChecker tests input availability checking
func TestInputChecker(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "string reader with content",
			input:    "test input",
			expected: true,
		},
		{
			name:     "string reader with empty content",
			input:    "",
			expected: true,
		},
		{
			name:     "string reader with JSON content",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: true,
		},
		{
			name:     "string reader with multiline content",
			input:    "line1\nline2\nline3",
			expected: true,
		},
	}

	ic := &inputChecker{}
	mockLogger := &MockLogger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result := ic.checkAvailability(mockLogger, reader)

			if result != tt.expected {
				t.Errorf("inputChecker.CheckAvailability() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPathParser tests path parsing functionality
func TestPathParser(t *testing.T) {
	pp := &pathParser{}

	tests := []struct {
		name     string
		input    string
		expected []string
		isJSON   bool
	}{
		{
			name:     "JSON input",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: []string{"/test.txt"},
			isJSON:   true,
		},
		{
			name:     "plain text input",
			input:    "/file1.txt\n/file2.txt",
			expected: []string{"/file1.txt", "/file2.txt"},
			isJSON:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pp.parse(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("pathParser.Parse() = %v, want %v", result, tt.expected)
			}

			isJSON := pp.isJSON(tt.input)
			if isJSON != tt.isJSON {
				t.Errorf("pathParser.IsJSON() = %v, want %v", isJSON, tt.isJSON)
			}
		})
	}
}

// TestInputReader tests the input reading functionality
func TestInputReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: []string{"/test.txt"},
		},
		{
			name:     "plain text input",
			input:    "/file1.txt\n/file2.txt",
			expected: []string{"/file1.txt", "/file2.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			mockLogger := &MockLogger{}
			ir := newInputReader()

			result := ir.readPaths(mockLogger, reader)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("InputReader.ReadPaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFileValidator tests file validation functionality
func TestFileValidator(t *testing.T) {
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
			name:     "non-empty file should be processed",
			filePath: nonEmptyFile,
			expected: true,
		},
		{
			name:     "empty file should not be processed",
			filePath: emptyFile,
			expected: false,
		},
		{
			name:     "non-existent file should not be processed",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
		},
	}

	fv := &fileValidator{}
	mockLogger := &MockLogger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fv.shouldProcess(tt.filePath, mockLogger)
			if result != tt.expected {
				t.Errorf("fileValidator.ShouldProcess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNewlineChecker tests newline checking functionality
func TestNewlineChecker(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		content       string
		expectedNeeds bool
		expectError   bool
	}{
		{
			name:          "file without newline",
			content:       "content",
			expectedNeeds: true,
			expectError:   false,
		},
		{
			name:          "file with newline",
			content:       "content\n",
			expectedNeeds: false,
			expectError:   false,
		},
	}

	nc := &newlineChecker{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, []byte(tt.content), 0o644)

			file, err := os.OpenFile(testFile, os.O_RDWR, 0o644)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			needsNewline, err := nc.needsNewline(file)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if needsNewline != tt.expectedNeeds {
				t.Errorf("newlineChecker.NeedsNewline() = %v, want %v", needsNewline, tt.expectedNeeds)
			}
		})
	}
}

// TestFileModifier tests file modification functionality
func TestFileModifier(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		initialContent  string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "add newline to content without newline",
			initialContent:  "content",
			expectedContent: "content\n",
			expectError:     false,
		},
		{
			name:            "add newline to empty content",
			initialContent:  "",
			expectedContent: "\n",
			expectError:     false,
		},
		{
			name:            "add newline to multiline content",
			initialContent:  "line1\nline2",
			expectedContent: "line1\nline2\n",
			expectError:     false,
		},
		{
			name:            "add newline to single character",
			initialContent:  "a",
			expectedContent: "a\n",
			expectError:     false,
		},
	}

	fm := &fileModifier{}
	mockLogger := &MockLogger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, []byte(tt.initialContent), 0o644)

			file, err := os.OpenFile(testFile, os.O_RDWR, 0o644)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			// Position the file pointer at the end for appending
			_, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				t.Fatalf("Failed to seek to end of file: %v", err)
			}

			err = fm.addNewline(file, testFile, mockLogger)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify content was modified correctly
				content, _ := os.ReadFile(testFile)
				if string(content) != tt.expectedContent {
					t.Errorf("Expected content %q, got %q", tt.expectedContent, string(content))
				}
			}
		})
	}
}

// TestFileProcessor tests the complete file processing pipeline
func TestFileProcessor(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		initialContent  string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "file without newline gets newline added",
			initialContent:  "content",
			expectedContent: "content\n",
			expectError:     false,
		},
		{
			name:            "file with newline remains unchanged",
			initialContent:  "content\n",
			expectedContent: "content\n",
			expectError:     false,
		},
	}

	fp := newFileProcessor()
	mockLogger := &MockLogger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, []byte(tt.initialContent), 0o644)

			err := fp.processFile(testFile, mockLogger)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			content, _ := os.ReadFile(testFile)
			if string(content) != tt.expectedContent {
				t.Errorf("Expected content %q, got %q", tt.expectedContent, string(content))
			}
		})
	}
}

// TestProgressLogger tests progress logging functionality
func TestProgressLogger(t *testing.T) {
	tests := []struct {
		name            string
		filePath        string
		current         int
		total           int
		expectedMessage string
	}{
		{
			name:            "first file of three",
			filePath:        "/test/file.txt",
			current:         1,
			total:           3,
			expectedMessage: "[1/3] Processing: /test/file.txt",
		},
		{
			name:            "single file",
			filePath:        "/single.txt",
			current:         1,
			total:           1,
			expectedMessage: "[1/1] Processing: /single.txt",
		},
		{
			name:            "last file of many",
			filePath:        "/final/file.txt",
			current:         10,
			total:           10,
			expectedMessage: "[10/10] Processing: /final/file.txt",
		},
		{
			name:            "long file path",
			filePath:        "/very/long/path/to/some/deeply/nested/file.txt",
			current:         2,
			total:           5,
			expectedMessage: "[2/5] Processing: /very/long/path/to/some/deeply/nested/file.txt",
		},
	}

	pl := &progressLogger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			pl.logProgress(mockLogger, tt.filePath, tt.current, tt.total)

			if len(mockLogger.DebugMessages) == 0 {
				t.Error("Expected debug message to be logged")
			} else if !strings.Contains(mockLogger.DebugMessages[0], tt.expectedMessage) {
				t.Errorf("Expected debug message to contain %q, got %q", tt.expectedMessage, mockLogger.DebugMessages[0])
			}
		})
	}
}

// TestErrorHandler tests error handling functionality
func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name             string
		filePath         string
		errorMsg         string
		expectedDebugMsg string
		expectedErrorMsg string
	}{
		{
			name:             "basic error handling",
			filePath:         "/test/file.txt",
			errorMsg:         "test error",
			expectedDebugMsg: "test error",
			expectedErrorMsg: "Error processing /test/file.txt: test error",
		},
		{
			name:             "permission error",
			filePath:         "/readonly/file.txt",
			errorMsg:         "permission denied",
			expectedDebugMsg: "permission denied",
			expectedErrorMsg: "Error processing /readonly/file.txt: permission denied",
		},
		{
			name:             "file not found error",
			filePath:         "/nonexistent/path.txt",
			errorMsg:         "file not found",
			expectedDebugMsg: "file not found",
			expectedErrorMsg: "Error processing /nonexistent/path.txt: file not found",
		},
		{
			name:             "long path error",
			filePath:         "/very/long/path/to/some/deeply/nested/file.txt",
			errorMsg:         "I/O error",
			expectedDebugMsg: "I/O error",
			expectedErrorMsg: "Error processing /very/long/path/to/some/deeply/nested/file.txt: I/O error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create errorHandler with buffer writer for testing
			var errorBuffer bytes.Buffer
			eh := &errorHandler{
				ErrorWriter: &errorBuffer,
			}
			mockLogger := &MockLogger{}

			// Call HandleError
			eh.handleError(mockLogger, tt.filePath, fmt.Errorf("%s", tt.errorMsg))

			// Check debug message was logged
			if len(mockLogger.DebugMessages) == 0 {
				t.Error("Expected debug message to be logged")
			} else if !strings.Contains(mockLogger.DebugMessages[0], tt.expectedDebugMsg) {
				t.Errorf("Expected debug message to contain %q, got %q", tt.expectedDebugMsg, mockLogger.DebugMessages[0])
			}

			// Check error writer output
			errorOutput := errorBuffer.String()
			if !strings.Contains(errorOutput, tt.expectedErrorMsg) {
				t.Errorf("Expected error output to contain %q, got %q", tt.expectedErrorMsg, errorOutput)
			}
		})
	}
}

// TestSingleFileProcessor tests single file processing functionality
func TestSingleFileProcessor(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		initialContent  string
		expectedContent string
		current         int
		total           int
		expectMessages  bool
	}{
		{
			name:            "process file without newline",
			initialContent:  "content",
			expectedContent: "content\n",
			current:         1,
			total:           1,
			expectMessages:  true,
		},
		{
			name:            "process file with newline",
			initialContent:  "content\n",
			expectedContent: "content\n",
			current:         2,
			total:           3,
			expectMessages:  true,
		},
		{
			name:            "process empty file",
			initialContent:  "",
			expectedContent: "",
			current:         1,
			total:           5,
			expectMessages:  true,
		},
		{
			name:            "process multiline file",
			initialContent:  "line1\nline2",
			expectedContent: "line1\nline2\n",
			current:         3,
			total:           4,
			expectMessages:  true,
		},
	}

	sfp := newSingleFileProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, []byte(tt.initialContent), 0o644)

			mockLogger := &MockLogger{}
			sfp.process(mockLogger, testFile, tt.current, tt.total)

			// Verify progress was logged
			if tt.expectMessages && len(mockLogger.DebugMessages) == 0 {
				t.Error("Expected debug messages to be logged")
			}

			// Verify file content
			content, _ := os.ReadFile(testFile)
			if string(content) != tt.expectedContent {
				t.Errorf("Expected content %q, got %q", tt.expectedContent, string(content))
			}

			// Verify progress message format
			if tt.expectMessages {
				expectedProgressMessage := fmt.Sprintf("[%d/%d] Processing:", tt.current, tt.total)
				found := false
				for _, msg := range mockLogger.DebugMessages {
					if strings.Contains(msg, expectedProgressMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected progress message containing %q in debug messages", expectedProgressMessage)
				}
			}
		})
	}
}

// TestDisplayStrategy tests the display strategy functionality
func TestTruncatedDisplayStrategy(t *testing.T) {
	tds := &truncatedDisplayStrategy{}
	mockLogger := &MockLogger{}

	tests := []struct {
		name     string
		lines    []string
		maxLines int
		expected int // expected number of debug messages
	}{
		{
			name:     "short lines display all",
			lines:    []string{"line1", "line2"},
			maxLines: 5,
			expected: 2,
		},
		{
			name:     "long lines get truncated",
			lines:    []string{"line1", "line2", "line3", "line4", "line5", "line6"},
			maxLines: 3,
			expected: 4, // 1 first + 1 omission + 2 last
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger.DebugMessages = nil // Reset
			tds.display(mockLogger, tt.lines, tt.maxLines)

			if len(mockLogger.DebugMessages) != tt.expected {
				t.Errorf("Expected %d debug messages, got %d", tt.expected, len(mockLogger.DebugMessages))
			}
		})
	}
}

// TestLineDisplayer tests the line displayer functionality
func TestLineDisplayer(t *testing.T) {
	tests := []struct {
		name              string
		lines             []string
		maxLines          int
		expectedMessages  int
		useCustomStrategy bool
	}{
		{
			name:              "display short lines",
			lines:             []string{"line1", "line2", "line3"},
			maxLines:          5,
			expectedMessages:  3,
			useCustomStrategy: false,
		},
		{
			name:              "display single line",
			lines:             []string{"single line"},
			maxLines:          3,
			expectedMessages:  1,
			useCustomStrategy: false,
		},
		{
			name:              "display empty lines",
			lines:             []string{},
			maxLines:          5,
			expectedMessages:  0,
			useCustomStrategy: false,
		},
		{
			name:              "display with custom strategy",
			lines:             []string{"line1", "line2"},
			maxLines:          4,
			expectedMessages:  2,
			useCustomStrategy: true,
		},
		{
			name:              "display many lines with truncation",
			lines:             []string{"line1", "line2", "line3", "line4", "line5", "line6"},
			maxLines:          3,
			expectedMessages:  4, // 1 first + 1 omission + 2 last
			useCustomStrategy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ld := newLineDisplayer()
			mockLogger := &MockLogger{}

			if tt.useCustomStrategy {
				customStrategy := &truncatedDisplayStrategy{}
				ld.setStrategy(customStrategy)
			}

			ld.display(mockLogger, tt.lines, tt.maxLines)

			if len(mockLogger.DebugMessages) != tt.expectedMessages {
				t.Errorf("Expected %d debug messages, got %d", tt.expectedMessages, len(mockLogger.DebugMessages))
			}

			// Verify line content appears in messages for non-empty cases
			if len(tt.lines) > 0 && tt.expectedMessages > 0 {
				found := false
				for _, msg := range mockLogger.DebugMessages {
					if strings.Contains(msg, tt.lines[0]) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected first line %q to appear in debug messages", tt.lines[0])
				}
			}
		})
	}
}

// TestTextParsers tests the text parser implementations
func TestJSONTextParser(t *testing.T) {
	jtp := &jsonTextParser{}

	tests := []struct {
		name     string
		input    string
		canParse bool
		expected []string
	}{
		{
			name:     "valid JSON",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			canParse: true,
			expected: []string{"/test.txt"},
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			canParse: false,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canParse := jtp.canParse(tt.input)
			if canParse != tt.canParse {
				t.Errorf("jsonTextParser.CanParse() = %v, want %v", canParse, tt.canParse)
			}

			result := jtp.parse(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("jsonTextParser.Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPlainTextParser tests plain text parsing
func TestPlainTextParser(t *testing.T) {
	ptp := &plainTextParser{}

	// plainTextParser should always return true for CanParse
	if !ptp.canParse("anything") {
		t.Error("plainTextParser.CanParse() should always return true")
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple lines",
			input:    "/file1.txt\n/file2.txt\n\n/file3.txt",
			expected: []string{"/file1.txt", "/file2.txt", "/file3.txt"},
		},
		{
			name:     "single line",
			input:    "/single.txt",
			expected: []string{"/single.txt"},
		},
		{
			name:     "empty lines ignored",
			input:    "\n\n/file.txt\n\n",
			expected: []string{"/file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ptp.parse(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("plainTextParser.Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCompositeTextParser tests the composite parser functionality
func TestCompositeTextParser(t *testing.T) {
	ctp := newCompositeTextParser()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON input parsed by JSON parser",
			input:    `{"tool_input": {"file_path": "/test.txt"}}`,
			expected: []string{"/test.txt"},
		},
		{
			name:     "plain text parsed by plain text parser",
			input:    "/file1.txt\n/file2.txt",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctp.parse(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CompositeTextParser.Parse() = %v, want %v", result, tt.expected)
			}
		})
	}

	// Test AddParser functionality
	customParser := &plainTextParser{}
	ctp.addParser(customParser)

	// Parser should still work after adding custom parser
	result := ctp.parse("/test.txt")
	expected := []string{"/test.txt"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("CompositeTextParser.Parse() after AddParser = %v, want %v", result, expected)
	}
}

// TestReadInputLinesTrailingEmptyLines tests readInputLines with trailing empty lines
func TestReadInputLinesTrailingEmptyLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "input with trailing empty lines",
			input:    "line1\nline2\n\n\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "input with single trailing empty line",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "input with only empty lines",
			input:    "\n\n\n",
			expected: nil,
		},
		{
			name:     "input with mixed empty lines",
			input:    "line1\n\nline2\n\n\n",
			expected: []string{"line1", "", "line2"},
		},
		{
			name:     "input without trailing newline",
			input:    "line1\nline2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "single line with trailing empty lines",
			input:    "single\n\n\n",
			expected: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result := readInputLines(reader)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("readInputLines() = %#v, want %#v", result, tt.expected)
			}
		})
	}
}

// TestCheckLastByteIOError tests checkLastByte with I/O errors
func TestCheckLastByteIOError(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name                 string
		fileContent          []byte
		expectedError        bool
		expectedNeedsNewline bool
	}{
		{
			name:                 "empty file should error on seek",
			fileContent:          []byte{},
			expectedError:        true,
			expectedNeedsNewline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")
			_ = os.WriteFile(testFile, tt.fileContent, 0o644)

			file, err := os.OpenFile(testFile, os.O_RDWR, 0o644)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			needsNewline, err := checkLastByte(file)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if needsNewline != tt.expectedNeedsNewline {
				t.Errorf("Expected needsNewline %v, got %v", tt.expectedNeedsNewline, needsNewline)
			}
		})
	}
}

// TestGlobPatternMatcher tests the glob pattern matching functionality
func TestGlobPatternMatcher(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		filePath string
		expected bool
	}{
		{
			name:     "no patterns - should not match",
			patterns: []string{},
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "single pattern exact match",
			patterns: []string{"*.txt"},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "single pattern no match",
			patterns: []string{"*.go"},
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "multiple patterns - first matches",
			patterns: []string{"*.txt", "*.md"},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "multiple patterns - second matches",
			patterns: []string{"*.go", "*.txt"},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "path with directory - full path match",
			patterns: []string{"src/*.go"},
			filePath: "src/main.go",
			expected: true,
		},
		{
			name:     "path with directory - basename match",
			patterns: []string{"*.go"},
			filePath: "src/main.go",
			expected: true,
		},
		{
			name:     "wildcard pattern",
			patterns: []string{"test_*"},
			filePath: "test_file.txt",
			expected: true,
		},
		{
			name:     "complex pattern no match",
			patterns: []string{"*.py", "test_*.md"},
			filePath: "main.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := newGlobPatternMatcher(tt.patterns)
			result := matcher.matches(tt.filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for patterns %v and path %s", tt.expected, result, tt.patterns, tt.filePath)
			}
		})
	}
}

// TestFileFilter tests the file filtering functionality
func TestFileFilter(t *testing.T) {
	tests := []struct {
		name     string
		exclude  []string
		include  []string
		filePath string
		expected bool
	}{
		{
			name:     "no patterns - should process",
			exclude:  []string{},
			include:  []string{},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "exclude pattern matches - should not process",
			exclude:  []string{"*.txt"},
			include:  []string{},
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "exclude pattern no match - should process",
			exclude:  []string{"*.go"},
			include:  []string{},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "include pattern matches - should process",
			exclude:  []string{},
			include:  []string{"*.txt"},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "include pattern no match - should not process",
			exclude:  []string{},
			include:  []string{"*.go"},
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "multiple exclude patterns - one matches",
			exclude:  []string{"*.go", "*.txt"},
			include:  []string{},
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "multiple include patterns - one matches",
			exclude:  []string{},
			include:  []string{"*.go", "*.txt"},
			filePath: "test.txt",
			expected: true,
		},
		{
			name:     "complex path with exclude",
			exclude:  []string{"temp/*"},
			include:  []string{},
			filePath: "temp/test.txt",
			expected: false,
		},
		{
			name:     "complex path with include",
			exclude:  []string{},
			include:  []string{"src/*"},
			filePath: "src/main.go",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config{
				Exclude: tt.exclude,
				Include: tt.include,
			}
			filter := newFileFilter(config)
			result := filter.shouldProcess(tt.filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for exclude %v, include %v, path %s",
					tt.expected, result, tt.exclude, tt.include, tt.filePath)
			}
		})
	}
}

// TestParseFlagsWithPatterns tests the pattern parsing functionality
func TestParseFlagsWithPatterns(t *testing.T) {
	// Save original command line args and restore at the end
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name            string
		args            []string
		expectedExclude []string
		expectedInclude []string
		expectExit      bool
	}{
		{
			name:            "no patterns",
			args:            []string{"ccnewline"},
			expectedExclude: nil,
			expectedInclude: nil,
			expectExit:      false,
		},
		{
			name:            "single exclude pattern",
			args:            []string{"ccnewline", "--exclude", "*.txt"},
			expectedExclude: []string{"*.txt"},
			expectedInclude: nil,
			expectExit:      false,
		},
		{
			name:            "single include pattern",
			args:            []string{"ccnewline", "--include", "*.go"},
			expectedExclude: nil,
			expectedInclude: []string{"*.go"},
			expectExit:      false,
		},
		{
			name:            "multiple exclude patterns",
			args:            []string{"ccnewline", "--exclude", "*.txt,*.md"},
			expectedExclude: []string{"*.txt", "*.md"},
			expectedInclude: nil,
			expectExit:      false,
		},
		{
			name:            "multiple include patterns with spaces",
			args:            []string{"ccnewline", "--include", "*.go, *.txt"},
			expectedExclude: nil,
			expectedInclude: []string{"*.go", "*.txt"},
			expectExit:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag package for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			os.Args = tt.args

			parser := newFlagParser()
			config := parser.parse()

			if !reflect.DeepEqual(config.Exclude, tt.expectedExclude) {
				t.Errorf("Expected exclude %v, got %v", tt.expectedExclude, config.Exclude)
			}
			if !reflect.DeepEqual(config.Include, tt.expectedInclude) {
				t.Errorf("Expected include %v, got %v", tt.expectedInclude, config.Include)
			}
		})
	}
}
