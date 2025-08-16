// Package cli provides command-line interface functionality for ccnewline.
// It handles configuration, flag parsing, version information, and argument validation.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Version information, set by goreleaser during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Config holds the configuration options for the tool
type Config struct {
	// Debug enables detailed processing information output
	Debug bool
	// Silent disables all output when processing files
	Silent bool
	// Exclude contains glob patterns for files to exclude from processing
	// Mutually exclusive with Include
	Exclude []string
	// Include contains glob patterns for files to include in processing
	// Mutually exclusive with Exclude
	Include []string
}

// IsDebugMode returns whether debug mode is enabled
func (c *Config) IsDebugMode() bool {
	return c.Debug
}

// IsSilent returns whether silent mode is enabled
func (c *Config) IsSilent() bool {
	return c.Silent
}

// versionHandler handles version display functionality
type versionHandler struct{}

// showVersion displays version information and exits
func (vh *versionHandler) showVersion() {
	fmt.Printf("ccnewline %s (Built on %s from Git SHA %s)\n", version, date, commit)
	os.Exit(0)
}

// argumentValidator validates command-line arguments
type argumentValidator struct{}

// validateArgs checks for conflicting arguments
func (av *argumentValidator) validateArgs(config *Config) {
	if len(config.Exclude) > 0 && len(config.Include) > 0 {
		fmt.Fprintf(os.Stderr, "Error: --exclude and --include are mutually exclusive\n")
		os.Exit(1)
	}
}

// flagParser handles command-line flag parsing
type flagParser struct {
	flagSet   *flag.FlagSet
	validator *argumentValidator
	vHandler  *versionHandler
}

// newFlagParser creates a new flag parser
func newFlagParser() *flagParser {
	return &flagParser{
		flagSet:   flag.NewFlagSet("ccnewline", flag.ExitOnError),
		validator: &argumentValidator{},
		vHandler:  &versionHandler{},
	}
}

// parse processes command-line arguments and returns configuration
func (fp *flagParser) parse() *Config {
	var config Config
	var showVersion bool
	var excludeStr, includeStr string

	fp.flagSet.Usage = usage
	defineBoolFlag(fp.flagSet, &config.Debug, "debug", "d", false, "Enable debug output")
	defineBoolFlag(fp.flagSet, &config.Silent, "silent", "s", false, "Silent mode - no output")
	defineBoolFlag(fp.flagSet, &showVersion, "version", "v", false, "Show version information")
	defineStringFlag(fp.flagSet, &excludeStr, "exclude", "e", "", "Exclude files matching glob patterns (comma-separated)")
	defineStringFlag(fp.flagSet, &includeStr, "include", "i", "", "Include only files matching glob patterns (comma-separated)")

	var showHelp bool
	defineBoolFlag(fp.flagSet, &showHelp, "help", "h", false, "Show this help message")

	_ = fp.flagSet.Parse(os.Args[1:])

	if showVersion {
		fp.vHandler.showVersion()
	}

	if showHelp {
		usage()
		os.Exit(0)
	}

	if excludeStr != "" {
		config.Exclude = parsePatterns(excludeStr)
	}
	if includeStr != "" {
		config.Include = parsePatterns(includeStr)
	}

	fp.validator.validateArgs(&config)
	return &config
}

// usage prints the program usage information
func usage() {
	fmt.Fprintf(os.Stderr, `ccnewline - Automatically adds newline characters to files processed by Claude Code hooks
Designed as a PostToolUse hook for Edit, MultiEdit, and Write tools.

Usage: %s [options] < input.json

Options:
  -d, --debug      Enable debug output
  -s, --silent     Silent mode - no output
  -v, --version    Show version information
  -h, --help       Show this help message
  -e, --exclude    Exclude files matching glob patterns (comma-separated)
  -i, --include    Include only files matching glob patterns (comma-separated)
`, os.Args[0])
}

// defineBoolFlag defines a boolean flag with both long and short forms
func defineBoolFlag(fs *flag.FlagSet, ptr *bool, name, shorthand string, value bool, usage string) {
	fs.BoolVar(ptr, name, value, usage)
	if shorthand != "" {
		fs.BoolVar(ptr, shorthand, value, usage+" (shorthand)")
	}
}

// defineStringFlag defines a string flag with both long and short forms
func defineStringFlag(fs *flag.FlagSet, ptr *string, name, shorthand string, value, usage string) {
	fs.StringVar(ptr, name, value, usage)
	if shorthand != "" {
		fs.StringVar(ptr, shorthand, value, usage+" (shorthand)")
	}
}

// parsePatterns splits comma-separated patterns into a slice
func parsePatterns(patterns string) []string {
	if patterns == "" {
		return nil
	}
	var result []string
	for pattern := range strings.SplitSeq(patterns, ",") {
		if trimmed := strings.TrimSpace(pattern); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParseFlags processes command-line arguments and returns configuration
func ParseFlags() *Config {
	parser := newFlagParser()
	return parser.parse()
}
