package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/filter"
	"github.com/StardustEnigma/gologify/pkg/output"
	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	analyzeFormat string
	analyzeLimit  int
	searchTerm    string
	filterExprs   []string
	levelFilter   string
	regexFilter   string
	timeFrom      string
	timeTo        string
	aggregate     bool
	groupBy       string
	topIPs        int
	topErrors     int
	outputFormat  string
	highlight     string
	workers       int
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "Analyze a log file",
	Long: `Parse and analyze a log file with support for searching, filtering,
aggregation, and multiple output formats.

Examples:
  gologify analyze app.log
  gologify analyze app.log --search "error" --format json
  gologify analyze app.log --filter "status:500" --aggregate
  gologify analyze access.log --group-by level --output table`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// Input format.
	analyzeCmd.Flags().StringVarP(&analyzeFormat, "format", "f", "auto", "log format: auto, json, text, csv, syslog")
	analyzeCmd.Flags().IntVarP(&analyzeLimit, "limit", "n", 0, "max entries to display (0 = unlimited)")

	// Search & filter.
	analyzeCmd.Flags().StringVarP(&searchTerm, "search", "s", "", "search for keyword across all fields")
	analyzeCmd.Flags().StringArrayVar(&filterExprs, "filter", nil, "field filter (e.g., status:500, level:ERROR)")
	analyzeCmd.Flags().StringVarP(&levelFilter, "level", "l", "", "filter by log level")
	analyzeCmd.Flags().StringVar(&regexFilter, "regex", "", "filter by regex on raw line")
	analyzeCmd.Flags().StringVar(&timeFrom, "from", "", "filter from time (RFC3339)")
	analyzeCmd.Flags().StringVar(&timeTo, "to", "", "filter until time (RFC3339)")

	// Aggregation.
	analyzeCmd.Flags().BoolVarP(&aggregate, "aggregate", "a", false, "show aggregated statistics")
	analyzeCmd.Flags().StringVarP(&groupBy, "group-by", "g", "", "group results by field")
	analyzeCmd.Flags().IntVar(&topIPs, "top-ips", 0, "show top N IPs")
	analyzeCmd.Flags().IntVar(&topErrors, "top-errors", 0, "show top N errors")

	// Output.
	analyzeCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, csv, raw")
	analyzeCmd.Flags().StringVar(&highlight, "highlight", "", "highlight matching text")

	// Performance.
	analyzeCmd.Flags().IntVar(&workers, "workers", 0, "concurrent workers (0 = auto)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Open and validate file.
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", filePath)
	}
	if info.Size() == 0 {
		fmt.Fprintln(os.Stderr, "file is empty")
		return nil
	}

	// Determine log format.
	format, err := parser.ParseFormat(analyzeFormat)
	if err != nil {
		return err
	}

	var reader io.Reader = file
	if format == parser.FormatAuto {
		detected, newReader := parser.DetectFormat(file)
		format = detected
		reader = newReader
		if verbose {
			fmt.Fprintf(os.Stderr, "auto-detected format: %s\n", format)
		}
	}

	// Build filter chain.
	chain, err := filter.BuildChain(searchTerm, filterExprs, levelFilter, regexFilter, timeFrom, timeTo)
	if err != nil {
		return err
	}

	// Decide if we need aggregation.
	needsAggregation := aggregate || groupBy != "" || topIPs > 0 || topErrors > 0

	// Create aggregator if needed.
	var agg *aggregator.Aggregator
	if needsAggregation {
		errN := topErrors
		if errN == 0 {
			errN = 10 // default top errors count
		}
		agg = aggregator.New(groupBy, topIPs, errN)
	}

	// Start parsing.
	p := parser.NewParser(format)
	entries, errs := p.Parse(reader)

	// Drain parse errors in background.
	var errCount int64
	go func() {
		for e := range errs {
			atomic.AddInt64(&errCount, 1)
			if verbose {
				fmt.Fprintf(os.Stderr, "warning: %v\n", e)
			}
			_ = e
		}
	}()

	// Process entries through filter → aggregator / output.
	var totalCount, matchCount int

	// For non-aggregation mode, we need formatters.
	var jsonFmt *output.JSONFormatter
	var csvEntries []parser.LogEntry
	var rawFmt *output.RawFormatter

	switch outputFormat {
	case "json":
		jsonFmt = output.NewJSONFormatter(os.Stdout)
	case "csv":
		// CSV needs to collect all entries to write headers first.
		csvEntries = make([]parser.LogEntry, 0)
	case "raw":
		rawFmt = output.NewRawFormatter(os.Stdout, highlight)
	}

	for entry := range entries {
		totalCount++

		// Apply filters.
		if !chain.IsEmpty() && !chain.Match(entry) {
			continue
		}

		matchCount++

		// Aggregation mode: feed to aggregator.
		if needsAggregation {
			agg.Add(entry)
			continue
		}

		// Limit check.
		if analyzeLimit > 0 && matchCount > analyzeLimit {
			break
		}

		// Stream output.
		switch outputFormat {
		case "json":
			if err := jsonFmt.FormatEntry(entry); err != nil {
				return fmt.Errorf("writing JSON: %w", err)
			}
		case "csv":
			csvEntries = append(csvEntries, entry)
		case "raw":
			rawFmt.FormatEntry(entry)
		default:
			printEntryColored(entry)
		}
	}

	// Finalize output.
	if needsAggregation {
		agg.SetTotal(totalCount)
		result := agg.Result()
		topErrs := agg.TopErrors()

		switch outputFormat {
		case "json":
			jf := output.NewJSONFormatter(os.Stdout)
			if err := jf.FormatResult(result, topErrs); err != nil {
				return fmt.Errorf("writing JSON result: %w", err)
			}
		case "csv":
			cf := output.NewCSVFormatter(os.Stdout)
			if err := cf.FormatResult(result, topErrs); err != nil {
				return fmt.Errorf("writing CSV result: %w", err)
			}
		default:
			tf := output.NewTableFormatter(os.Stdout)
			tf.FormatResult(result, topErrs)
		}
	} else if outputFormat == "csv" && len(csvEntries) > 0 {
		cf := output.NewCSVFormatter(os.Stdout)
		if err := cf.FormatEntries(csvEntries); err != nil {
			return fmt.Errorf("writing CSV: %w", err)
		}
	}

	// Summary on stderr (only for table/raw output).
	if !needsAggregation && (outputFormat == "table" || outputFormat == "raw") {
		ec := atomic.LoadInt64(&errCount)
		fmt.Fprintf(os.Stderr, "\n%s %d entries displayed",
			color.CyanString("→"),
			matchCount,
		)
		if totalCount != matchCount {
			fmt.Fprintf(os.Stderr, " (of %d total)", totalCount)
		}
		if ec > 0 {
			fmt.Fprintf(os.Stderr, " (%d parse warnings)", ec)
		}
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

// printEntryColored renders a single log entry with colors to stdout.
func printEntryColored(entry parser.LogEntry) {
	var parts []string

	// Timestamp.
	if !entry.Timestamp.IsZero() {
		parts = append(parts, color.CyanString(entry.Timestamp.Format("2006-01-02 15:04:05")))
	}

	// Level.
	if entry.Level != "" {
		var lvl string
		switch entry.Level {
		case "ERROR", "FATAL", "PANIC":
			lvl = color.RedString("%-5s", entry.Level)
		case "WARN":
			lvl = color.YellowString("%-5s", entry.Level)
		case "INFO":
			lvl = color.GreenString("%-5s", entry.Level)
		case "DEBUG":
			lvl = color.CyanString("%-5s", entry.Level)
		default:
			lvl = fmt.Sprintf("%-5s", entry.Level)
		}
		parts = append(parts, lvl)
	}

	// Source.
	if entry.Source != "" {
		parts = append(parts, color.New(color.Bold).Sprintf("[%s]", entry.Source))
	}

	// Message.
	parts = append(parts, entry.Message)

	fmt.Println(strings.Join(parts, " "))
}
