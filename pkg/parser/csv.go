package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// CSVParser parses CSV-formatted log files. The first row is treated as
// a header and used to populate LogEntry.Fields. Common column names
// (timestamp, level, message) are mapped to the corresponding LogEntry fields.
type CSVParser struct{}

// Parse reads CSV records and emits LogEntry values. The first row must
// be a header row.
func (p *CSVParser) Parse(reader io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry, 256)
	errs := make(chan error, 64)

	go func() {
		defer close(entries)
		defer close(errs)

		csvReader := csv.NewReader(reader)
		csvReader.LazyQuotes = true
		csvReader.TrimLeadingSpace = true
		csvReader.ReuseRecord = false

		// Read header row.
		headers, err := csvReader.Read()
		if err != nil {
			errs <- fmt.Errorf("reading CSV header: %w", err)
			return
		}

		// Normalize headers for field matching.
		normalizedHeaders := make([]string, len(headers))
		for i, h := range headers {
			normalizedHeaders[i] = strings.ToLower(strings.TrimSpace(h))
		}

		// Identify well-known columns by position.
		tsCol := findColumnIndex(normalizedHeaders, "timestamp", "time", "ts", "date", "datetime", "@timestamp")
		levelCol := findColumnIndex(normalizedHeaders, "level", "severity", "loglevel", "lvl")
		msgCol := findColumnIndex(normalizedHeaders, "message", "msg", "text", "description")
		sourceCol := findColumnIndex(normalizedHeaders, "source", "service", "host", "hostname", "app")

		lineNum := 1 // header was line 1
		for {
			lineNum++
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errs <- fmt.Errorf("line %d: %w", lineNum, err)
				continue
			}

			entry := LogEntry{
				Raw:     strings.Join(record, ","),
				LineNum: lineNum,
				Fields:  make(map[string]string),
			}

			// Map known columns.
			if tsCol >= 0 && tsCol < len(record) {
				if t, ok := tryParseTimestamp(record[tsCol]); ok {
					entry.Timestamp = t
				}
			}
			if levelCol >= 0 && levelCol < len(record) {
				entry.Level = NormalizeLevel(record[levelCol])
			}
			if msgCol >= 0 && msgCol < len(record) {
				entry.Message = record[msgCol]
			}
			if sourceCol >= 0 && sourceCol < len(record) {
				entry.Source = record[sourceCol]
			}

			// All columns go into Fields keyed by their original header name.
			for i, val := range record {
				if i < len(headers) {
					entry.Fields[headers[i]] = val
				}
			}

			entries <- entry
		}
	}()

	return entries, errs
}

// findColumnIndex returns the index of the first header that matches any
// of the candidate names, or -1 if none match.
func findColumnIndex(headers []string, candidates ...string) int {
	for i, h := range headers {
		for _, c := range candidates {
			if h == c {
				return i
			}
		}
	}
	return -1
}
