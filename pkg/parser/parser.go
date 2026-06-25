// Package parser provides streaming log file parsers for multiple formats.
//
// All parsers implement the Parser interface and emit LogEntry values
// through a channel, enabling constant-memory processing of arbitrarily
// large files.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

// LogEntry is the universal representation of a single log line,
// regardless of the original format. Fields that cannot be extracted
// from a given format are left at their zero values.
type LogEntry struct {
	Timestamp time.Time         // parsed timestamp, zero if unparseable
	Level     string            // normalized log level (INFO, WARN, ERROR, DEBUG, FATAL)
	Message   string            // the log message body
	Source    string            // originating source (hostname, service, filename)
	Fields    map[string]string // arbitrary key-value pairs extracted from the log
	Raw       string            // the original, unmodified line
	LineNum   int               // 1-based line number in the source file
}

// Parser defines the interface that all log format parsers must implement.
// Parse reads from the provided reader and streams LogEntry values through
// the returned channel. The error channel receives any non-fatal parse errors.
// Both channels are closed when the reader is exhausted.
type Parser interface {
	Parse(reader io.Reader) (<-chan LogEntry, <-chan error)
}

// Format enumerates the supported log formats.
type Format string

const (
	FormatAuto   Format = "auto"
	FormatJSON   Format = "json"
	FormatText   Format = "text"
	FormatCSV    Format = "csv"
	FormatSyslog Format = "syslog"
)

// ParseFormat converts a user-provided format string to a Format constant.
// Returns an error if the format is unrecognized.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "":
		return FormatAuto, nil
	case "json":
		return FormatJSON, nil
	case "text", "plain", "txt":
		return FormatText, nil
	case "csv":
		return FormatCSV, nil
	case "syslog":
		return FormatSyslog, nil
	default:
		return "", fmt.Errorf("unsupported log format %q (supported: auto, json, text, csv, syslog)", s)
	}
}

// DetectFormat samples the first few lines of the reader to guess the log
// format. It returns the detected Format and a new reader that replays the
// sampled bytes followed by the remainder of the original reader.
func DetectFormat(reader io.Reader) (Format, io.Reader) {
	const sampleLines = 10
	buf := bufio.NewReader(reader)

	var lines []string
	for i := 0; i < sampleLines; i++ {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
		if err != nil {
			break
		}
	}

	// Reconstruct a reader that includes the bytes we consumed.
	combined := io.MultiReader(
		strings.NewReader(strings.Join(lines, "\n")+"\n"),
		buf,
	)

	if len(lines) == 0 {
		return FormatText, combined
	}

	// Heuristic 1: if most lines start with '{', it's JSON lines.
	jsonCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "{") {
			jsonCount++
		}
	}
	if jsonCount > len(lines)/2 {
		return FormatJSON, combined
	}

	// Heuristic 2: if the first line looks like a CSV header (has commas, no spaces before commas).
	if strings.Count(lines[0], ",") >= 2 && !strings.HasPrefix(lines[0], "<") {
		allHaveCommas := true
		commaCount := strings.Count(lines[0], ",")
		for _, l := range lines[1:] {
			if strings.Count(l, ",") != commaCount {
				allHaveCommas = false
				break
			}
		}
		if allHaveCommas {
			return FormatCSV, combined
		}
	}

	// Heuristic 3: syslog typically starts with '<' priority or month abbreviation.
	syslogCount := 0
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for _, l := range lines {
		if strings.HasPrefix(l, "<") {
			syslogCount++
			continue
		}
		for _, m := range months {
			if strings.HasPrefix(l, m) {
				syslogCount++
				break
			}
		}
	}
	if syslogCount > len(lines)/2 {
		return FormatSyslog, combined
	}

	// Default: plain text.
	return FormatText, combined
}

// NewParser creates a parser for the given format.
func NewParser(format Format) Parser {
	switch format {
	case FormatJSON:
		return &JSONParser{}
	case FormatCSV:
		return &CSVParser{}
	case FormatSyslog:
		return &SyslogParser{}
	default:
		return &TextParser{}
	}
}

// NormalizeLevel normalizes common log level strings to a canonical form.
func NormalizeLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "TRACE", "TRC":
		return "TRACE"
	case "DEBUG", "DBG", "DEBU":
		return "DEBUG"
	case "INFO", "INF", "INFORMATION":
		return "INFO"
	case "WARN", "WRN", "WARNING":
		return "WARN"
	case "ERROR", "ERR":
		return "ERROR"
	case "FATAL", "FTL", "CRITICAL", "CRIT":
		return "FATAL"
	case "PANIC":
		return "PANIC"
	default:
		return strings.ToUpper(strings.TrimSpace(level))
	}
}

// tryParseTimestamp attempts to parse a timestamp string against a set of
// common formats, returning the first successful parse.
func tryParseTimestamp(s string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"02/Jan/2006:15:04:05 -0700",
		"Jan  2 15:04:05",
		"Jan 2 15:04:05",
		"2006/01/02 15:04:05",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
	}
	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
