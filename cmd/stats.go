package cmd

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/filter"
	"github.com/StardustEnigma/gologify/pkg/output"
	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	statsFormat    string
	statsTopIPs    int
	statsTopErrors int
	statsSearch    string
	statsLevel     string
	statsOutput    string
)

var statsCmd = &cobra.Command{
	Use:   "stats [file]",
	Short: "Show quick statistics for a log file",
	Long: `Display summary statistics for a log file including level distribution,
time range, top errors, and top IPs.

Examples:
  gologify stats app.log
  gologify stats access.log --top-ips 10
  gologify stats app.log --top-errors 5 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().StringVarP(&statsFormat, "format", "f", "auto", "log format: auto, json, text, csv, syslog")
	statsCmd.Flags().IntVar(&statsTopIPs, "top-ips", 10, "show top N IPs")
	statsCmd.Flags().IntVar(&statsTopErrors, "top-errors", 10, "show top N errors")
	statsCmd.Flags().StringVarP(&statsSearch, "search", "s", "", "filter by keyword before computing stats")
	statsCmd.Flags().StringVarP(&statsLevel, "level", "l", "", "filter by log level before computing stats")
	statsCmd.Flags().StringVarP(&statsOutput, "output", "o", "table", "output format: table, json, csv")
}

func runStats(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close file: %v\n", err)
		}
	}()

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

	// Determine format.
	format, err := parser.ParseFormat(statsFormat)
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

	// Build optional filter.
	chain, err := filter.BuildChain(statsSearch, nil, statsLevel, "", "", "")
	if err != nil {
		return err
	}

	// Create aggregator with stats defaults.
	agg := aggregator.New("", statsTopIPs, statsTopErrors)

	// Parse and aggregate.
	p := parser.NewParser(format)
	entries, errs := p.Parse(reader)

	var errCount int64
	go func() {
		for range errs {
			atomic.AddInt64(&errCount, 1)
		}
	}()

	totalCount := 0
	for entry := range entries {
		totalCount++
		if !chain.IsEmpty() && !chain.Match(entry) {
			continue
		}
		agg.Add(entry)
	}

	agg.SetTotal(totalCount)
	result := agg.Result()
	topErrs := agg.TopErrors()

	// Output results.
	switch statsOutput {
	case "json":
		jf := output.NewJSONFormatter(os.Stdout)
		return jf.FormatResult(result, topErrs)
	case "csv":
		cf := output.NewCSVFormatter(os.Stdout)
		return cf.FormatResult(result, topErrs)
	default:
		tf := output.NewTableFormatter(os.Stdout)
		tf.FormatResult(result, topErrs)

		ec := atomic.LoadInt64(&errCount)
		if ec > 0 {
			fmt.Fprintf(os.Stderr, "%s %d parse warnings\n", color.YellowString("⚠"), ec)
		}
	}

	return nil
}
