package logging

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/koh-sh/ccnewline/internal/cli"
)

func TestConsoleLogger(t *testing.T) {
	tests := []struct {
		name           string
		config         *cli.Config
		logMessage     string
		debugMessage   string
		expectedStdout string
		expectedStderr string
	}{
		{
			name: "normal mode",
			config: &cli.Config{
				Debug:  false,
				Silent: false,
			},
			logMessage:     "test log",
			debugMessage:   "test debug",
			expectedStdout: "test log\n",
			expectedStderr: "",
		},
		{
			name: "debug mode",
			config: &cli.Config{
				Debug:  true,
				Silent: false,
			},
			logMessage:     "test log",
			debugMessage:   "test debug",
			expectedStdout: "test log\n",
			expectedStderr: "test debug\n",
		},
		{
			name: "silent mode",
			config: &cli.Config{
				Debug:  false,
				Silent: true,
			},
			logMessage:     "test log",
			debugMessage:   "test debug",
			expectedStdout: "",
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			logger := NewConsoleLogger(tt.config)

			logger.Info(tt.logMessage)
			logger.Debug(tt.debugMessage)

			wOut.Close()
			wErr.Close()

			stdout, _ := io.ReadAll(rOut)
			stderr, _ := io.ReadAll(rErr)

			if string(stdout) != tt.expectedStdout {
				t.Errorf("Expected stdout %q, got %q", tt.expectedStdout, string(stdout))
			}

			if string(stderr) != tt.expectedStderr {
				t.Errorf("Expected stderr %q, got %q", tt.expectedStderr, string(stderr))
			}
		})
	}
}

func TestTruncateInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short input",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "long input",
			input:    "line1\nline2\nline3\nline4\nline5",
			expected: "...\nline3\nline4\nline5",
		},
		{
			name:     "exactly 3 lines",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShowInputDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := &testLogger{output: &buf}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short input",
			input:    "line1\nline2",
			expected: "Raw input:\n  line1\n  line2\n\n",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			ShowInputDebug(logger, tt.input)

			if buf.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestConsoleLoggerError(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	config := &cli.Config{Debug: false, Silent: false}
	logger := NewConsoleLogger(config)

	logger.Error("test error message")

	w.Close()
	stderr, _ := io.ReadAll(r)

	expected := "test error message\n"
	if string(stderr) != expected {
		t.Errorf("Expected stderr %q, got %q", expected, string(stderr))
	}
}

func TestConsoleLoggerLogFileProcessing(t *testing.T) {
	tests := []struct {
		name           string
		config         *cli.Config
		file           string
		result         string
		expectedStdout string
		expectedStderr string
	}{
		{
			name: "debug mode",
			config: &cli.Config{
				Debug:  true,
				Silent: false,
			},
			file:           "test.txt",
			result:         "Added newline",
			expectedStdout: "",
			expectedStderr: "  test.txt: Added newline\n",
		},
		{
			name: "normal mode with added newline",
			config: &cli.Config{
				Debug:  false,
				Silent: false,
			},
			file:           "test.txt",
			result:         "Added newline",
			expectedStdout: "Added newline to test.txt\n",
			expectedStderr: "",
		},
		{
			name: "normal mode with no change",
			config: &cli.Config{
				Debug:  false,
				Silent: false,
			},
			file:           "test.txt",
			result:         "No change",
			expectedStdout: "",
			expectedStderr: "",
		},
		{
			name: "silent mode",
			config: &cli.Config{
				Debug:  false,
				Silent: true,
			},
			file:           "test.txt",
			result:         "Added newline",
			expectedStdout: "",
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			logger := NewConsoleLogger(tt.config)
			logger.LogFileProcessing(tt.file, tt.result)

			wOut.Close()
			wErr.Close()

			stdout, _ := io.ReadAll(rOut)
			stderr, _ := io.ReadAll(rErr)

			if string(stdout) != tt.expectedStdout {
				t.Errorf("Expected stdout %q, got %q", tt.expectedStdout, string(stdout))
			}

			if string(stderr) != tt.expectedStderr {
				t.Errorf("Expected stderr %q, got %q", tt.expectedStderr, string(stderr))
			}
		})
	}
}

func TestConsoleLoggerShowProcessingStart(t *testing.T) {
	tests := []struct {
		name         string
		config       *cli.Config
		files        []string
		expectOutput bool
	}{
		{
			name: "debug mode with files",
			config: &cli.Config{
				Debug:  true,
				Silent: false,
			},
			files:        []string{"file1.txt", "file2.go"},
			expectOutput: true,
		},
		{
			name: "debug mode with no files",
			config: &cli.Config{
				Debug:  true,
				Silent: false,
			},
			files:        []string{},
			expectOutput: true,
		},
		{
			name: "normal mode",
			config: &cli.Config{
				Debug:  false,
				Silent: false,
			},
			files:        []string{"file1.txt"},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := NewConsoleLogger(tt.config)
			logger.ShowProcessingStart(tt.files)

			w.Close()
			stderr, _ := io.ReadAll(r)

			if tt.expectOutput {
				if len(stderr) == 0 {
					t.Error("Expected debug output but got none")
				}
				// Check that it contains expected elements
				output := string(stderr)
				if !bytes.Contains(stderr, []byte("INPUT")) {
					t.Error("Expected 'INPUT' section in debug output")
				}
				if len(tt.files) > 0 {
					for _, file := range tt.files {
						if !bytes.Contains(stderr, []byte(file)) {
							t.Errorf("Expected file %s in debug output: %s", file, output)
						}
					}
				} else if !bytes.Contains(stderr, []byte("No files to process")) {
					t.Error("Expected 'No files to process' message")
				}
			} else if len(stderr) > 0 {
				t.Errorf("Expected no output but got: %s", string(stderr))
			}
		})
	}
}

func TestConsoleLoggerShowProcessingEnd(t *testing.T) {
	tests := []struct {
		name           string
		config         *cli.Config
		totalFiles     int
		processedFiles int
		expectOutput   bool
	}{
		{
			name: "debug mode",
			config: &cli.Config{
				Debug:  true,
				Silent: false,
			},
			totalFiles:     5,
			processedFiles: 3,
			expectOutput:   true,
		},
		{
			name: "normal mode",
			config: &cli.Config{
				Debug:  false,
				Silent: false,
			},
			totalFiles:     5,
			processedFiles: 3,
			expectOutput:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := NewConsoleLogger(tt.config)
			logger.ShowProcessingEnd(tt.totalFiles, tt.processedFiles)

			w.Close()
			stderr, _ := io.ReadAll(r)

			if tt.expectOutput {
				if len(stderr) == 0 {
					t.Error("Expected debug output but got none")
				}
				output := string(stderr)
				if !bytes.Contains(stderr, []byte("SUMMARY")) {
					t.Error("Expected 'SUMMARY' section in debug output")
				}
				if !bytes.Contains(stderr, []byte("Total files: 5")) {
					t.Errorf("Expected total files count in output: %s", output)
				}
				if !bytes.Contains(stderr, []byte("Files processed: 3")) {
					t.Errorf("Expected processed files count in output: %s", output)
				}
			} else if len(stderr) > 0 {
				t.Errorf("Expected no output but got: %s", string(stderr))
			}
		})
	}
}

// testLogger implements Logger interface for testing
type testLogger struct {
	output io.Writer
}

func (tl *testLogger) Debug(message string) {
	_, _ = tl.output.Write([]byte(message + "\n"))
}

func (tl *testLogger) Info(message string) {
	_, _ = tl.output.Write([]byte(message + "\n"))
}

func (tl *testLogger) Error(message string) {
	_, _ = tl.output.Write([]byte(message + "\n"))
}

func (tl *testLogger) LogFileProcessing(file string, result string) {
	_, _ = tl.output.Write([]byte(file + ": " + result + "\n"))
}

func (tl *testLogger) ShowProcessingStart(files []string) {
	_, _ = tl.output.Write([]byte("Processing start\n"))
}

func (tl *testLogger) ShowProcessingEnd(totalFiles, processedFiles int) {
	_, _ = tl.output.Write([]byte("Processing end\n"))
}
