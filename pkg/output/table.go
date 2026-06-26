// Package output provides formatters for rendering log entries and
// aggregation results in various formats (table, JSON, CSV, raw).
package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// TableFormatter renders log entries and aggregation results as
// pretty-printed terminal tables with color-coded log levels.
type TableFormatter struct {
	writer io.Writer
}

// NewTableFormatter creates a table formatter writing to the given writer.
func NewTableFormatter(w io.Writer) *TableFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &TableFormatter{writer: w}
}

// newMinimalTable creates a borderless, clean table matching the old style.
func newMinimalTable(w io.Writer) *tablewriter.Table {
	return tablewriter.NewTable(w,
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{
				Left:   tw.Off,
				Right:  tw.Off,
				Top:    tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Separators: tw.Separators{
					ShowHeader:     tw.Off,
					ShowFooter:     tw.Off,
					BetweenRows:    tw.Off,
					BetweenColumns: tw.Off,
				},
				Lines: tw.Lines{
					ShowTop:        tw.Off,
					ShowBottom:     tw.Off,
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.Off,
				},
			},
		}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithPadding(tw.Padding{Left: "  ", Right: ""}),
	)
}

// FormatEntries writes log entries as a formatted table.
func (f *TableFormatter) FormatEntries(entries []parser.LogEntry) {
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No entries to display.")
		return
	}

	table := newMinimalTable(f.writer)
	table.Header([]string{"#", "Timestamp", "Level", "Source", "Message"})

	for _, entry := range entries {
		ts := ""
		if !entry.Timestamp.IsZero() {
			ts = entry.Timestamp.Format("15:04:05")
		}

		level := colorLevel(entry.Level)
		source := entry.Source
		msg := entry.Message
		if len(msg) > 120 {
			msg = msg[:117] + "..."
		}

		_ = table.Append([]string{
			fmt.Sprintf("%d", entry.LineNum),
			ts,
			level,
			source,
			msg,
		})
	}

	_ = table.Render()
}

// FormatResult writes aggregation results as formatted tables.
func (f *TableFormatter) FormatResult(result aggregator.Result, topErrors []aggregator.TopEntry) {
	// Summary section.
	_, _ = fmt.Fprintln(f.writer)
	headerColor := color.New(color.FgCyan, color.Bold)
	_, _ = headerColor.Fprintln(f.writer, "═══ Log Analysis Summary ═══")
	_, _ = fmt.Fprintln(f.writer)

	_, _ = fmt.Fprintf(f.writer, "  %s %d\n", color.CyanString("Matched Entries:"), result.MatchedEntries)
	if result.TotalEntries > 0 {
		_, _ = fmt.Fprintf(f.writer, "  %s %d\n", color.CyanString("Total Entries:  "), result.TotalEntries)
	}
	if result.FirstTimestamp != "" {
		_, _ = fmt.Fprintf(f.writer, "  %s %s → %s\n",
			color.CyanString("Time Range:     "),
			result.FirstTimestamp, result.LastTimestamp)
	}

	// Level distribution.
	if len(result.LevelCounts) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = headerColor.Fprintln(f.writer, "─── Level Distribution ───")
		_, _ = fmt.Fprintln(f.writer)

		table := newMinimalTable(f.writer)
		table.Header([]string{"Level", "Count", "Bar"})

		// Sort levels by priority.
		levels := sortLevels(result.LevelCounts)
		maxCount := 0
		for _, c := range result.LevelCounts {
			if c > maxCount {
				maxCount = c
			}
		}

		for _, level := range levels {
			count := result.LevelCounts[level]
			barLen := 0
			if maxCount > 0 {
				barLen = (count * 30) / maxCount
			}
			if barLen == 0 && count > 0 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen)
			_ = table.Append([]string{colorLevel(level), fmt.Sprintf("%d", count), bar})
		}
		_ = table.Render()
	}

	// Group-by results.
	for field, counts := range result.GroupCounts {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = headerColor.Fprintf(f.writer, "─── Grouped by: %s ───\n", field)
		_, _ = fmt.Fprintln(f.writer)

		table := newMinimalTable(f.writer)
		table.Header([]string{tw.Title(field), "Count"})

		sorted := sortedByCount(counts)
		for _, e := range sorted {
			_ = table.Append([]string{e.Value, fmt.Sprintf("%d", e.Count)})
		}
		_ = table.Render()
	}

	// Top N entries (IPs).
	if len(result.TopN) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = headerColor.Fprintln(f.writer, "─── Top IPs ───")
		_, _ = fmt.Fprintln(f.writer)

		table := newMinimalTable(f.writer)
		table.Header([]string{"Rank", "IP", "Requests"})

		for i, e := range result.TopN {
			_ = table.Append([]string{fmt.Sprintf("%d", i+1), e.Value, fmt.Sprintf("%d", e.Count)})
		}
		_ = table.Render()
	}

	// Top errors.
	if len(topErrors) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = headerColor.Fprintln(f.writer, "─── Top Errors ───")
		_, _ = fmt.Fprintln(f.writer)

		table := newMinimalTable(f.writer)
		table.Header([]string{"Rank", "Error", "Count"})

		for i, e := range topErrors {
			msg := e.Value
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}
			_ = table.Append([]string{fmt.Sprintf("%d", i+1), msg, fmt.Sprintf("%d", e.Count)})
		}
		_ = table.Render()
	}

	// Numeric stats.
	if len(result.NumericStats) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = headerColor.Fprintln(f.writer, "─── Numeric Fields ───")
		_, _ = fmt.Fprintln(f.writer)

		table := newMinimalTable(f.writer)
		table.Header([]string{"Field", "Count", "Min", "Max", "Avg", "Sum"})

		fields := make([]string, 0, len(result.NumericStats))
		for k := range result.NumericStats {
			fields = append(fields, k)
		}
		sort.Strings(fields)

		for _, field := range fields {
			stat := result.NumericStats[field]
			_ = table.Append([]string{
				field,
				fmt.Sprintf("%d", stat.Count),
				fmt.Sprintf("%.2f", stat.Min),
				fmt.Sprintf("%.2f", stat.Max),
				fmt.Sprintf("%.2f", stat.Avg),
				fmt.Sprintf("%.2f", stat.Sum),
			})
		}
		_ = table.Render()
	}

	_, _ = fmt.Fprintln(f.writer)
}

// colorLevel returns a color-coded level string for terminal display.
func colorLevel(level string) string {
	switch level {
	case "ERROR", "FATAL", "PANIC":
		return color.RedString("%-5s", level)
	case "WARN":
		return color.YellowString("%-5s", level)
	case "INFO":
		return color.GreenString("%-5s", level)
	case "DEBUG":
		return color.CyanString("%-5s", level)
	case "TRACE":
		return fmt.Sprintf("%-5s", level)
	default:
		return fmt.Sprintf("%-5s", level)
	}
}

// sortLevels returns levels sorted by severity priority.
func sortLevels(counts map[string]int) []string {
	priority := map[string]int{
		"FATAL": 6, "PANIC": 5, "ERROR": 4, "WARN": 3,
		"INFO": 2, "DEBUG": 1, "TRACE": 0,
	}
	levels := make([]string, 0, len(counts))
	for k := range counts {
		levels = append(levels, k)
	}
	sort.Slice(levels, func(i, j int) bool {
		pi, oki := priority[levels[i]]
		pj, okj := priority[levels[j]]
		if !oki {
			pi = -1
		}
		if !okj {
			pj = -1
		}
		return pi > pj
	})
	return levels
}

// sortedByCount sorts a map by count descending.
func sortedByCount(counts map[string]int) []aggregator.TopEntry {
	entries := make([]aggregator.TopEntry, 0, len(counts))
	for k, v := range counts {
		entries = append(entries, aggregator.TopEntry{Value: k, Count: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})
	return entries
}
