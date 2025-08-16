package processing

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koh-sh/ccnewline/internal/cli"
)

func TestNeedsNewlineFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: true,
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
			name:     "non-existing file",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			expected: false,
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
			name:     "non-existing file",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			expected: true,
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

func TestGlobPatternMatcher(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		expected bool
	}{
		{
			name:     "no patterns",
			patterns: []string{},
			path:     "test.txt",
			expected: false,
		},
		{
			name:     "matching pattern",
			patterns: []string{"*.txt"},
			path:     "test.txt",
			expected: true,
		},
		{
			name:     "non-matching pattern",
			patterns: []string{"*.go"},
			path:     "test.txt",
			expected: false,
		},
		{
			name:     "multiple patterns with match",
			patterns: []string{"*.go", "*.txt"},
			path:     "test.txt",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := newGlobPatternMatcher(tt.patterns)
			result := matcher.matches(tt.path)
			if result != tt.expected {
				t.Errorf("matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFileFilter(t *testing.T) {
	tests := []struct {
		name     string
		config   *cli.Config
		filePath string
		expected bool
	}{
		{
			name: "no filters",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{},
			},
			filePath: "test.txt",
			expected: true,
		},
		{
			name: "exclude filter matches",
			config: &cli.Config{
				Exclude: []string{"*.txt"},
				Include: []string{},
			},
			filePath: "test.txt",
			expected: false,
		},
		{
			name: "include filter matches",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{"*.txt"},
			},
			filePath: "test.txt",
			expected: true,
		},
		{
			name: "include filter doesn't match",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{"*.go"},
			},
			filePath: "test.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newFileFilter(tt.config)
			result := filter.shouldProcess(tt.filePath)
			if result != tt.expected {
				t.Errorf("shouldProcess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCheckLastByte(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		content     []byte
		expectNeed  bool
		expectError bool
	}{
		{
			name:        "file with newline",
			content:     []byte("hello\n"),
			expectNeed:  false,
			expectError: false,
		},
		{
			name:        "file without newline",
			content:     []byte("hello"),
			expectNeed:  true,
			expectError: false,
		},
		{
			name:        "empty file",
			content:     []byte{},
			expectNeed:  true,
			expectError: false,
		},
		{
			name:        "single newline",
			content:     []byte("\n"),
			expectNeed:  false,
			expectError: false,
		},
		{
			name:        "multiline with newline",
			content:     []byte("line1\nline2\n"),
			expectNeed:  false,
			expectError: false,
		},
		{
			name:        "multiline without newline",
			content:     []byte("line1\nline2"),
			expectNeed:  true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.name+".txt")
			err := os.WriteFile(filePath, tt.content, 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			needsNewline, err := checkLastByte(filePath)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if needsNewline != tt.expectNeed {
				t.Errorf("checkLastByte() = %v, want %v", needsNewline, tt.expectNeed)
			}
		})
	}
}

func TestCheckLastByteWithNonExistentFile(t *testing.T) {
	_, err := checkLastByte("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestAddNewlineToFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		initialContent []byte
		expectContent  []byte
	}{
		{
			name:           "add newline to file without one",
			initialContent: []byte("hello"),
			expectContent:  []byte("hello\n"),
		},
		{
			name:           "add newline to empty file",
			initialContent: []byte{},
			expectContent:  []byte("\n"),
		},
		{
			name:           "add newline to file with existing newline",
			initialContent: []byte("hello\n"),
			expectContent:  []byte("hello\n\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.name+".txt")
			err := os.WriteFile(filePath, tt.initialContent, 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			err = addNewlineToFile(filePath)
			if err != nil {
				t.Errorf("addNewlineToFile() error = %v", err)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if !bytes.Equal(content, tt.expectContent) {
				t.Errorf("File content = %q, want %q", content, tt.expectContent)
			}
		})
	}
}

func TestAddNewlineToFileWithNonExistentFile(t *testing.T) {
	err := addNewlineToFile("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
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

func TestProcessFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "test1.txt")
	file2 := filepath.Join(tempDir, "test2.go")
	file3 := filepath.Join(tempDir, "test3.md")

	_ = os.WriteFile(file1, []byte("content1"), 0o644)
	_ = os.WriteFile(file2, []byte("content2\n"), 0o644)
	_ = os.WriteFile(file3, []byte("content3"), 0o644)

	tests := []struct {
		name          string
		config        *cli.Config
		filePaths     []string
		expectedCount int
		expectedSkips int
	}{
		{
			name: "process all files",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{},
			},
			filePaths:     []string{file1, file2, file3},
			expectedCount: 3,
			expectedSkips: 0,
		},
		{
			name: "exclude .txt files",
			config: &cli.Config{
				Exclude: []string{"*.txt"},
				Include: []string{},
			},
			filePaths:     []string{file1, file2, file3},
			expectedCount: 2,
			expectedSkips: 1,
		},
		{
			name: "include only .go files",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{"*.go"},
			},
			filePaths:     []string{file1, file2, file3},
			expectedCount: 1,
			expectedSkips: 2,
		},
		{
			name: "empty file list",
			config: &cli.Config{
				Exclude: []string{},
				Include: []string{},
			},
			filePaths:     []string{},
			expectedCount: 0,
			expectedSkips: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			filter := newFileFilter(tt.config)

			processedCount := ProcessFiles(logger, tt.filePaths, filter)

			if processedCount != tt.expectedCount {
				t.Errorf("ProcessFiles() = %v, want %v", processedCount, tt.expectedCount)
			}

			actualSkips := 0
			for _, msg := range logger.debugMessages {
				if strings.Contains(msg, "Skipping") {
					actualSkips++
				}
			}
			if actualSkips != tt.expectedSkips {
				t.Errorf("Expected %d skips, got %d", tt.expectedSkips, actualSkips)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("content"), 0o644)

	tests := []struct {
		name         string
		config       *cli.Config
		input        string
		expectCalled bool
	}{
		{
			name: "run with json input",
			config: &cli.Config{
				Debug:   false,
				Silent:  true,
				Exclude: []string{},
				Include: []string{},
			},
			input:        `{"tool_input": {"file_path": "` + testFile + `"}}`,
			expectCalled: true,
		},
		{
			name: "run with empty input",
			config: &cli.Config{
				Debug:   false,
				Silent:  true,
				Exclude: []string{},
				Include: []string{},
			},
			input:        "",
			expectCalled: true,
		},
		{
			name: "run with plain text input",
			config: &cli.Config{
				Debug:   false,
				Silent:  true,
				Exclude: []string{},
				Include: []string{},
			},
			input:        testFile,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			input := strings.NewReader(tt.input)

			Run(tt.config, logger, input)

			if tt.expectCalled {
				found := false
				for _, msg := range logger.debugMessages {
					if strings.Contains(msg, "Processing") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected processing to be called")
				}
			}
		})
	}
}

func TestProcessSingleFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		content        []byte
		expectModified bool
	}{
		{
			name:           "file needs newline",
			content:        []byte("content"),
			expectModified: true,
		},
		{
			name:           "file has newline",
			content:        []byte{99, 111, 110, 116, 101, 110, 116, 10},
			expectModified: false,
		},
		{
			name:           "empty file",
			content:        []byte{},
			expectModified: false, // Empty files are skipped by shouldProcessFile
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.name+".txt")
			err := os.WriteFile(filePath, tt.content, 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			originalContent, _ := os.ReadFile(filePath)
			logger := &mockLogger{}

			err = processSingleFile(logger, filePath)
			if err != nil {
				t.Errorf("processSingleFile() error = %v", err)
			}

			newContent, _ := os.ReadFile(filePath)
			modified := !bytes.Equal(originalContent, newContent)

			if modified != tt.expectModified {
				t.Errorf("File modification = %v, want %v", modified, tt.expectModified)
			}

			if tt.expectModified {
				// Should end with newline after processing
				if len(newContent) == 0 || newContent[len(newContent)-1] != 10 {
					t.Error("File should end with newline after processing")
				}
			}
		})
	}
}

func TestProcessSingleFileWithNonExistentFile(t *testing.T) {
	logger := &mockLogger{}
	err := processSingleFile(logger, "/non/existent/file.txt")
	// Should not return error for non-existent file (just skip processing)
	if err != nil {
		t.Errorf("Unexpected error for non-existent file: %v", err)
	}
}
