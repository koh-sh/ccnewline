// Package main provides a command-line tool for automatically adding newline characters
// to files processed by Claude Code hooks. It is designed as a PostToolUse hook to
// ensure files modified by Edit, MultiEdit, and Write tools end with proper newlines.
package main

import (
	"os"

	"github.com/koh-sh/ccnewline/internal/cli"
	"github.com/koh-sh/ccnewline/internal/logging"
	"github.com/koh-sh/ccnewline/internal/processing"
)

// Version information, set by goreleaser during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// main is the entry point of the ccnewline tool
func main() {
	config := cli.ParseFlags(version, commit, date)
	logger := logging.NewConsoleLogger(config)
	processing.Run(config, logger, os.Stdin)
}
