package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// JSONParser parses JSON Lines (one JSON object per line), the standard
// format for structured logging libraries like zerolog, zap, and logrus.
type JSONParser struct{}

// Parse reads JSON lines from the reader and emits LogEntry values.
// Malformed lines are reported as errors but do not stop processing.
func (p *JSONParser) Parse(reader io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry, 256)
	errs := make(chan error, 64)

	go func() {
		defer close(entries)
		defer close(errs)

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			raw := scanner.Text()
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}

			var fields map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
				errs <- fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
				// Still emit the raw line as a basic entry so nothing is silently lost.
				entries <- LogEntry{Raw: raw, LineNum: lineNum, Message: raw}
				continue
			}

			entry := LogEntry{
				Raw:     raw,
				LineNum: lineNum,
				Fields:  make(map[string]string),
			}

			// Extract well-known fields, mapping common naming conventions.
			entry.Timestamp = extractTimestamp(fields)
			entry.Level = extractLevel(fields)
			entry.Message = extractMessage(fields)
			entry.Source = extractStringField(fields, "source", "service", "hostname", "host", "app")

			// Everything else goes into Fields.
			knownKeys := map[string]bool{
				"timestamp": true, "time": true, "ts": true, "t": true, "@timestamp": true,
				"level": true, "severity": true, "lvl": true, "loglevel": true,
				"message": true, "msg": true, "text": true,
				"source": true, "service": true, "hostname": true, "host": true, "app": true,
			}
			for k, v := range fields {
				if knownKeys[strings.ToLower(k)] {
					continue
				}
				entry.Fields[k] = fmt.Sprintf("%v", v)
			}

			entries <- entry
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("reading input: %w", err)
		}
	}()

	return entries, errs
}

// extractTimestamp looks for common timestamp field names and parses the value.
func extractTimestamp(fields map[string]interface{}) time.Time {
	keys := []string{"timestamp", "time", "ts", "t", "@timestamp"}
	for _, k := range keys {
		v, ok := caseInsensitiveLookup(fields, k)
		if !ok {
			continue
		}

		switch val := v.(type) {
		case string:
			if t, ok := tryParseTimestamp(val); ok {
				return t
			}
		case float64:
			// Epoch seconds (possibly with fractional milliseconds).
			sec := int64(val)
			nsec := int64((val - float64(sec)) * 1e9)
			return time.Unix(sec, nsec)
		}
	}
	return time.Time{}
}

// extractLevel looks for common level field names and normalizes the value.
func extractLevel(fields map[string]interface{}) string {
	keys := []string{"level", "severity", "lvl", "loglevel"}
	for _, k := range keys {
		v, ok := caseInsensitiveLookup(fields, k)
		if !ok {
			continue
		}
		return NormalizeLevel(fmt.Sprintf("%v", v))
	}
	return ""
}

// extractMessage looks for common message field names.
func extractMessage(fields map[string]interface{}) string {
	keys := []string{"message", "msg", "text"}
	for _, k := range keys {
		v, ok := caseInsensitiveLookup(fields, k)
		if !ok {
			continue
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// extractStringField returns the first found value among the candidate keys.
func extractStringField(fields map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := caseInsensitiveLookup(fields, k)
		if ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// caseInsensitiveLookup finds a key in the map regardless of case.
func caseInsensitiveLookup(fields map[string]interface{}, key string) (interface{}, bool) {
	// Try exact match first (fast path).
	if v, ok := fields[key]; ok {
		return v, true
	}
	// Fall back to case-insensitive scan.
	lower := strings.ToLower(key)
	for k, v := range fields {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return nil, false
}
