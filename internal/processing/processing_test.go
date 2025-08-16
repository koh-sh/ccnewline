package processing

import (
	"os"
	"path/filepath"
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
