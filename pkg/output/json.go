package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/parser"
)

// JSONFormatter renders log entries and aggregation results as JSON.
// Entries are written as JSON Lines (one object per line) for streaming.
// Aggregation results are written as a single JSON object.
type JSONFormatter struct {
	writer  io.Writer
	encoder *json.Encoder
}

// NewJSONFormatter creates a JSON formatter writing to the given writer.
func NewJSONFormatter(w io.Writer) *JSONFormatter {
	if w == nil {
		w = os.Stdout
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &JSONFormatter{writer: w, encoder: enc}
}

// jsonEntry is the JSON representation of a log entry.
type jsonEntry struct {
	Line      int               `json:"line"`
	Timestamp string            `json:"timestamp,omitempty"`
	Level     string            `json:"level,omitempty"`
	Source    string            `json:"source,omitempty"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// FormatEntry writes a single log entry as a JSON line.
func (f *JSONFormatter) FormatEntry(entry parser.LogEntry) error {
	je := jsonEntry{
		Line:    entry.LineNum,
		Level:   entry.Level,
		Source:  entry.Source,
		Message: entry.Message,
		Fields:  entry.Fields,
	}
	if !entry.Timestamp.IsZero() {
		je.Timestamp = entry.Timestamp.Format("2006-01-02T15:04:05Z07:00")
	}
	if len(je.Fields) == 0 {
		je.Fields = nil
	}
	return f.encoder.Encode(je)
}

// FormatEntries writes multiple log entries as JSON lines.
func (f *JSONFormatter) FormatEntries(entries []parser.LogEntry) error {
	for _, entry := range entries {
		if err := f.FormatEntry(entry); err != nil {
			return fmt.Errorf("encoding entry: %w", err)
		}
	}
	return nil
}

// jsonResult is the JSON representation of an aggregation result.
type jsonResult struct {
	Summary      jsonSummary                     `json:"summary"`
	LevelCounts  map[string]int                  `json:"level_counts,omitempty"`
	GroupCounts   map[string]map[string]int       `json:"group_counts,omitempty"`
	NumericStats map[string]*aggregator.NumericStat `json:"numeric_stats,omitempty"`
	TopIPs       []aggregator.TopEntry           `json:"top_ips,omitempty"`
	TopErrors    []aggregator.TopEntry           `json:"top_errors,omitempty"`
}

type jsonSummary struct {
	MatchedEntries int    `json:"matched_entries"`
	TotalEntries   int    `json:"total_entries,omitempty"`
	FirstTimestamp string `json:"first_timestamp,omitempty"`
	LastTimestamp   string `json:"last_timestamp,omitempty"`
}

// FormatResult writes an aggregation result as a single JSON object.
func (f *JSONFormatter) FormatResult(result aggregator.Result, topErrors []aggregator.TopEntry) error {
	jr := jsonResult{
		Summary: jsonSummary{
			MatchedEntries: result.MatchedEntries,
			TotalEntries:   result.TotalEntries,
			FirstTimestamp: result.FirstTimestamp,
			LastTimestamp:   result.LastTimestamp,
		},
		LevelCounts:  result.LevelCounts,
		NumericStats: result.NumericStats,
		TopIPs:       result.TopN,
		TopErrors:    topErrors,
	}

	if len(result.GroupCounts) > 0 {
		jr.GroupCounts = result.GroupCounts
	}
	if len(jr.LevelCounts) == 0 {
		jr.LevelCounts = nil
	}
	if len(jr.NumericStats) == 0 {
		jr.NumericStats = nil
	}

	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(jr)
}
