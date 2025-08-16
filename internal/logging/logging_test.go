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
