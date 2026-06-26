package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/parser"
)

// CSVFormatter renders log entries and aggregation results as CSV.
type CSVFormatter struct {
	writer    io.Writer
	csvWriter *csv.Writer
}

// NewCSVFormatter creates a CSV formatter writing to the given writer.
func NewCSVFormatter(w io.Writer) *CSVFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &CSVFormatter{
		writer:    w,
		csvWriter: csv.NewWriter(w),
	}
}

// FormatEntries writes log entries as CSV rows with a header.
func (f *CSVFormatter) FormatEntries(entries []parser.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Collect all unique field keys across entries.
	fieldKeys := collectFieldKeys(entries)

	// Write header.
	header := []string{"line", "timestamp", "level", "source", "message"}
	header = append(header, fieldKeys...)
	if err := f.csvWriter.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write rows.
	for _, entry := range entries {
		ts := ""
		if !entry.Timestamp.IsZero() {
			ts = entry.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		}

		row := []string{
			fmt.Sprintf("%d", entry.LineNum),
			ts,
			entry.Level,
			entry.Source,
			entry.Message,
		}

		for _, k := range fieldKeys {
			row = append(row, entry.Fields[k])
		}

		if err := f.csvWriter.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	f.csvWriter.Flush()
	return f.csvWriter.Error()
}

// FormatResult writes aggregation results as CSV.
func (f *CSVFormatter) FormatResult(result aggregator.Result, topErrors []aggregator.TopEntry) error {
	// Level counts.
	if len(result.LevelCounts) > 0 {
		if err := f.csvWriter.Write([]string{"level", "count"}); err != nil {
			return err
		}
		for level, count := range result.LevelCounts {
			if err := f.csvWriter.Write([]string{level, fmt.Sprintf("%d", count)}); err != nil {
				return err
			}
		}
		if err := f.csvWriter.Write([]string{}); err != nil {
			return err
		} // blank line separator
	}

	// Group counts.
	for field, counts := range result.GroupCounts {
		if err := f.csvWriter.Write([]string{field, "count"}); err != nil {
			return err
		}
		for value, count := range counts {
			if err := f.csvWriter.Write([]string{value, fmt.Sprintf("%d", count)}); err != nil {
				return err
			}
		}
		if err := f.csvWriter.Write([]string{}); err != nil {
			return err
		}
	}

	// Top IPs.
	if len(result.TopN) > 0 {
		if err := f.csvWriter.Write([]string{"ip", "requests"}); err != nil {
			return err
		}
		for _, e := range result.TopN {
			if err := f.csvWriter.Write([]string{e.Value, fmt.Sprintf("%d", e.Count)}); err != nil {
				return err
			}
		}
		if err := f.csvWriter.Write([]string{}); err != nil {
			return err
		}
	}

	// Top errors.
	if len(topErrors) > 0 {
		if err := f.csvWriter.Write([]string{"error", "count"}); err != nil {
			return err
		}
		for _, e := range topErrors {
			if err := f.csvWriter.Write([]string{e.Value, fmt.Sprintf("%d", e.Count)}); err != nil {
				return err
			}
		}
	}

	f.csvWriter.Flush()
	return f.csvWriter.Error()
}

// collectFieldKeys returns all unique field keys across entries, sorted.
func collectFieldKeys(entries []parser.LogEntry) []string {
	keys := make(map[string]bool)
	for _, e := range entries {
		for k := range e.Fields {
			keys[k] = true
		}
	}

	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
