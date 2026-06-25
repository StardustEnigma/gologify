package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// TextParser parses unstructured plain-text log files by matching lines
// against a set of common log patterns. Lines that don't match any
// pattern are treated as raw messages.
type TextParser struct{}

// Common log line patterns, ordered from most specific to least.
// Each pattern captures: timestamp, level, and message.
var textPatterns = []*regexp.Regexp{
	// [2024-01-15T10:30:45Z] [ERROR] message
	regexp.MustCompile(`^\[(?P<timestamp>\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[^\]]*)\]\s*\[(?P<level>\w+)\]\s*(?P<message>.*)$`),

	// 2024-01-15 10:30:45.123 ERROR [source] message
	regexp.MustCompile(`^(?P<timestamp>\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[.,]?\d*)\s+(?P<level>TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|PANIC|CRIT(?:ICAL)?)\s+(?:\[(?P<source>[^\]]+)\]\s+)?(?P<message>.*)$`),

	// 2024-01-15 10:30:45 - ERROR - message (Python style)
	regexp.MustCompile(`^(?P<timestamp>\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[.,]?\d*)\s+-\s+(?P<level>\w+)\s+-\s+(?P<message>.*)$`),

	// ERROR 2024-01-15 10:30:45 message (level first)
	regexp.MustCompile(`^(?P<level>TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|PANIC|CRIT(?:ICAL)?)\s+(?P<timestamp>\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[.,]?\d*)\s+(?P<message>.*)$`),

	// timestamp message (no level)
	regexp.MustCompile(`^(?P<timestamp>\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[.,]?\d*)\s+(?P<message>.*)$`),
}

// Parse reads plain text log lines and attempts to extract structured
// fields using pattern matching.
func (p *TextParser) Parse(reader io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry, 256)
	errs := make(chan error, 64)

	go func() {
		defer close(entries)
		defer close(errs)

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			raw := scanner.Text()
			if strings.TrimSpace(raw) == "" {
				continue
			}

			entry := LogEntry{
				Raw:     raw,
				LineNum: lineNum,
				Fields:  make(map[string]string),
			}

			matched := false
			for _, pattern := range textPatterns {
				match := pattern.FindStringSubmatch(raw)
				if match == nil {
					continue
				}

				for i, name := range pattern.SubexpNames() {
					if i == 0 || name == "" || match[i] == "" {
						continue
					}
					switch name {
					case "timestamp":
						if t, ok := tryParseTimestamp(match[i]); ok {
							entry.Timestamp = t
						}
					case "level":
						entry.Level = NormalizeLevel(match[i])
					case "message":
						entry.Message = match[i]
					case "source":
						entry.Source = match[i]
					default:
						entry.Fields[name] = match[i]
					}
				}
				matched = true
				break
			}

			// If no pattern matched, use the entire line as the message.
			if !matched {
				entry.Message = raw
			}

			entries <- entry
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("reading input: %w", err)
		}
	}()

	return entries, errs
}
