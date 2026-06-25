package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

// Global flags shared across all subcommands.
var (
	verbose bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "gologify",
	Short: "A fast CLI tool for log analysis and aggregation",
	Long: `GoLogify is a command-line tool for parsing, searching, filtering,
aggregating, and exporting log files. Built for DevOps engineers,
SREs, and developers who need fast log analysis from the terminal.

Supports JSON, plain text, CSV, and syslog formats with streaming
processing for files of any size.`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			color.NoColor = true
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	rootCmd.SetVersionTemplate(fmt.Sprintf("GoLogify version %s\n", Version))
}

// Execute runs the root command. Called from main.
func Execute() error {
	return rootCmd.Execute()
}
