package cmd

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/StardustEnigma/gologify/pkg/filter"
	"github.com/StardustEnigma/gologify/pkg/output"
	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	exportOutput    string
	exportFormat    string
	exportLogFormat string
	exportSearch    string
	exportFilter    []string
	exportLevel     string
	exportRegex     string
	exportFrom      string
	exportTo        string
	exportLimit     int
)

var exportCmd = &cobra.Command{
	Use:   "export [input-file]",
	Short: "Export filtered log entries to a file",
	Long: `Read a log file, apply filters, and export matching entries
to a new file in the specified format.

Examples:
  gologify export app.log --output errors.json --format json --search "ERROR"
  gologify export access.log --output filtered.csv --format csv --filter "status:500"
  gologify export app.log --output recent.log --format raw --from "2024-01-15T08:00:00Z"`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (required)")
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "export format: json, csv, raw")
	exportCmd.Flags().StringVar(&exportLogFormat, "log-format", "auto", "input log format: auto, json, text, csv, syslog")
	exportCmd.Flags().StringVarP(&exportSearch, "search", "s", "", "filter by keyword")
	exportCmd.Flags().StringArrayVar(&exportFilter, "filter", nil, "field filter (e.g., status:500)")
	exportCmd.Flags().StringVarP(&exportLevel, "level", "l", "", "filter by log level")
	exportCmd.Flags().StringVar(&exportRegex, "regex", "", "filter by regex")
	exportCmd.Flags().StringVar(&exportFrom, "from", "", "filter from time (RFC3339)")
	exportCmd.Flags().StringVar(&exportTo, "to", "", "filter until time (RFC3339)")
	exportCmd.Flags().IntVarP(&exportLimit, "limit", "n", 0, "max entries to export (0 = unlimited)")

	_ = exportCmd.MarkFlagRequired("output")
}

func runExport(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Open input file.
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer inFile.Close()

	info, err := inFile.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat input file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", inputPath)
	}
	if info.Size() == 0 {
		return fmt.Errorf("input file is empty")
	}

	// Open output file.
	outFile, err := os.Create(exportOutput)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer outFile.Close()

	// Determine log format.
	format, err := parser.ParseFormat(exportLogFormat)
	if err != nil {
		return err
	}

	var reader io.Reader = inFile
	if format == parser.FormatAuto {
		detected, newReader := parser.DetectFormat(inFile)
		format = detected
		reader = newReader
		if verbose {
			fmt.Fprintf(os.Stderr, "auto-detected format: %s\n", format)
		}
	}

	// Build filter chain.
	chain, err := filter.BuildChain(exportSearch, exportFilter, exportLevel, exportRegex, exportFrom, exportTo)
	if err != nil {
		return err
	}

	// Parse.
	p := parser.NewParser(format)
	entries, errs := p.Parse(reader)

	var errCount int64
	go func() {
		for range errs {
			atomic.AddInt64(&errCount, 1)
		}
	}()

	// Collect entries (needed for CSV headers).
	var matched []parser.LogEntry
	totalCount := 0

	for entry := range entries {
		totalCount++

		if !chain.IsEmpty() && !chain.Match(entry) {
			continue
		}

		matched = append(matched, entry)

		if exportLimit > 0 && len(matched) >= exportLimit {
			break
		}
	}

	// Write output.
	switch exportFormat {
	case "json":
		jf := output.NewJSONFormatter(outFile)
		if err := jf.FormatEntries(matched); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	case "csv":
		cf := output.NewCSVFormatter(outFile)
		if err := cf.FormatEntries(matched); err != nil {
			return fmt.Errorf("writing CSV: %w", err)
		}
	case "raw":
		rf := output.NewRawFormatter(outFile, "")
		rf.FormatEntries(matched)
	default:
		return fmt.Errorf("unsupported export format %q (supported: json, csv, raw)", exportFormat)
	}

	// Report.
	ec := atomic.LoadInt64(&errCount)
	fmt.Fprintf(os.Stderr, "%s exported %d entries to %s (%s format)\n",
		color.GreenString("✓"),
		len(matched),
		exportOutput,
		exportFormat,
	)
	if totalCount != len(matched) {
		fmt.Fprintf(os.Stderr, "  %d of %d total entries matched filters\n", len(matched), totalCount)
	}
	if ec > 0 {
		fmt.Fprintf(os.Stderr, "  %d parse warnings\n", ec)
	}

	return nil
}
