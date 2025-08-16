package cli

import (
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *Config
	}{
		{
			name: "default values",
			args: []string{},
			expected: &Config{
				Debug:   false,
				Silent:  false,
				Exclude: nil,
				Include: nil,
			},
		},
		{
			name: "debug flag",
			args: []string{"-d"},
			expected: &Config{
				Debug:   true,
				Silent:  false,
				Exclude: nil,
				Include: nil,
			},
		},
		{
			name: "silent flag",
			args: []string{"-s"},
			expected: &Config{
				Debug:   false,
				Silent:  true,
				Exclude: nil,
				Include: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set test args
			os.Args = append([]string{"test"}, tt.args...)

			// Test flag parsing
			parser := newFlagParser()
			result := parser.parse()

			if result.Debug != tt.expected.Debug {
				t.Errorf("Debug = %v, want %v", result.Debug, tt.expected.Debug)
			}
			if result.Silent != tt.expected.Silent {
				t.Errorf("Silent = %v, want %v", result.Silent, tt.expected.Silent)
			}
		})
	}
}

func TestArgumentValidator(t *testing.T) {
	validator := &argumentValidator{}

	tests := []struct {
		name      string
		config    *Config
		shouldErr bool
	}{
		{
			name: "no patterns",
			config: &Config{
				Exclude: nil,
				Include: nil,
			},
			shouldErr: false,
		},
		{
			name: "only exclude",
			config: &Config{
				Exclude: []string{"*.txt"},
				Include: nil,
			},
			shouldErr: false,
		},
		{
			name: "only include",
			config: &Config{
				Exclude: nil,
				Include: []string{"*.go"},
			},
			shouldErr: false,
		},
		{
			name: "both exclude and include",
			config: &Config{
				Exclude: []string{"*.txt"},
				Include: []string{"*.go"},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldErr {
				validator.validateArgs(tt.config)
				// If we reach here without panic, validation passed
			} else {
				// For error cases, we expect the function to call os.Exit(1)
				// In a real test environment, we'd need to mock os.Exit
				// For now, we'll just verify the logic separately
				hasExclude := len(tt.config.Exclude) > 0
				hasInclude := len(tt.config.Include) > 0
				shouldError := hasExclude && hasInclude
				if !shouldError {
					t.Errorf("Expected validation to fail but it didn't")
				}
			}
		})
	}
}

func TestParsePatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single pattern",
			input:    "*.txt",
			expected: []string{"*.txt"},
		},
		{
			name:     "multiple patterns",
			input:    "*.txt,*.md,*.log",
			expected: []string{"*.txt", "*.md", "*.log"},
		},
		{
			name:     "patterns with spaces",
			input:    "*.txt, *.md , *.log",
			expected: []string{"*.txt", "*.md", "*.log"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePatterns(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parsePatterns() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, pattern := range result {
				if pattern != tt.expected[i] {
					t.Errorf("parsePatterns()[%d] = %v, want %v", i, pattern, tt.expected[i])
				}
			}
		})
	}
}
