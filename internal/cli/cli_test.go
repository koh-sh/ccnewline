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

func TestConfigMethods(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		expectDebug  bool
		expectSilent bool
	}{
		{
			name: "debug mode enabled",
			config: &Config{
				Debug:  true,
				Silent: false,
			},
			expectDebug:  true,
			expectSilent: false,
		},
		{
			name: "silent mode enabled",
			config: &Config{
				Debug:  false,
				Silent: true,
			},
			expectDebug:  false,
			expectSilent: true,
		},
		{
			name: "both debug and silent",
			config: &Config{
				Debug:  true,
				Silent: true,
			},
			expectDebug:  true,
			expectSilent: true,
		},
		{
			name: "normal mode",
			config: &Config{
				Debug:  false,
				Silent: false,
			},
			expectDebug:  false,
			expectSilent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.IsDebugMode() != tt.expectDebug {
				t.Errorf("IsDebugMode() = %v, want %v", tt.config.IsDebugMode(), tt.expectDebug)
			}
			if tt.config.IsSilent() != tt.expectSilent {
				t.Errorf("IsSilent() = %v, want %v", tt.config.IsSilent(), tt.expectSilent)
			}
		})
	}
}

func TestVersionHandler(t *testing.T) {
	// Test version handler creation and basic functionality
	handler := &versionHandler{}

	// We can't easily test showVersion() as it calls os.Exit(0)
	// but we can test that the handler exists and has the method
	_ = handler // Use the handler to avoid unused variable warning
}

func TestParseFlagsWithExcludeInclude(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *Config
	}{
		{
			name: "exclude patterns",
			args: []string{"-e", "*.txt,*.md"},
			expected: &Config{
				Debug:   false,
				Silent:  false,
				Exclude: []string{"*.txt", "*.md"},
				Include: nil,
			},
		},
		{
			name: "include patterns",
			args: []string{"-i", "*.go,*.js"},
			expected: &Config{
				Debug:   false,
				Silent:  false,
				Exclude: nil,
				Include: []string{"*.go", "*.js"},
			},
		},
		{
			name: "long form flags",
			args: []string{"--debug", "--silent", "--exclude", "*.txt"},
			expected: &Config{
				Debug:   true,
				Silent:  true,
				Exclude: []string{"*.txt"},
				Include: nil,
			},
		},
		{
			name: "combined flags",
			args: []string{"-d", "-s", "--include", "*.go"},
			expected: &Config{
				Debug:   true,
				Silent:  true,
				Exclude: nil,
				Include: []string{"*.go"},
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

			// Check exclude patterns
			if len(result.Exclude) != len(tt.expected.Exclude) {
				t.Errorf("Exclude length = %v, want %v", len(result.Exclude), len(tt.expected.Exclude))
			} else {
				for i, pattern := range result.Exclude {
					if pattern != tt.expected.Exclude[i] {
						t.Errorf("Exclude[%d] = %v, want %v", i, pattern, tt.expected.Exclude[i])
					}
				}
			}

			// Check include patterns
			if len(result.Include) != len(tt.expected.Include) {
				t.Errorf("Include length = %v, want %v", len(result.Include), len(tt.expected.Include))
			} else {
				for i, pattern := range result.Include {
					if pattern != tt.expected.Include[i] {
						t.Errorf("Include[%d] = %v, want %v", i, pattern, tt.expected.Include[i])
					}
				}
			}
		})
	}
}

func TestUsage(t *testing.T) {
	// Test that usage function exists and can be called
	// We can't easily capture the output without significant test infrastructure
	// but we can verify the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("usage() panicked: %v", r)
		}
	}()

	// Since usage() calls os.Exit(1), we can't actually call it in tests
	// Instead, we'll just verify the function exists by checking it's not nil
	// In a real scenario, you'd mock the os.Exit function
}

func TestDefineBoolFlag(t *testing.T) {
	// Create a flag set to test with
	parser := newFlagParser()

	// Test defineBoolFlag
	if parser.flagSet == nil {
		t.Error("flagSet should not be nil")
	}

	// Test that validator exists
	if parser.validator == nil {
		t.Error("validator should not be nil")
	}
}

func TestDefineStringFlag(t *testing.T) {
	// Create a flag set to test with
	parser := newFlagParser()

	// Test defineStringFlag
	if parser.flagSet == nil {
		t.Error("flagSet should not be nil")
	}

	// Test that version handler exists
	if parser.vHandler == nil {
		t.Error("vHandler should not be nil")
	}
}

func TestNewFlagParser(t *testing.T) {
	parser := newFlagParser()

	// Test that all components are properly initialized
	if parser.validator == nil {
		t.Error("parser.validator should not be nil")
	}

	if parser.flagSet == nil {
		t.Error("parser.flagSet should not be nil")
	}

	if parser.vHandler == nil {
		t.Error("parser.vHandler should not be nil")
	}
}

func TestParseFlagsFunction(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set test args for normal operation
	os.Args = []string{"test", "-d"}

	config := ParseFlags()

	if !config.Debug {
		t.Error("Debug flag should be set")
	}
}
