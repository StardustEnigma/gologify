package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SyslogParser parses RFC 3164 (BSD syslog) formatted log lines.
// These are the most common syslog format, as seen in /var/log/syslog
// and /var/log/messages.
type SyslogParser struct{}

// RFC 3164 format: <priority>Mon DD HH:MM:SS hostname process[pid]: message
// Also handles the variant without priority: Mon DD HH:MM:SS hostname process[pid]: message
var (
	syslogWithPriority = regexp.MustCompile(
		`^<(?P<priority>\d{1,3})>(?P<timestamp>\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<hostname>\S+)\s+(?P<process>[^\[:]+)(?:\[(?P<pid>\d+)\])?:\s*(?P<message>.*)$`,
	)
	syslogWithoutPriority = regexp.MustCompile(
		`^(?P<timestamp>\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<hostname>\S+)\s+(?P<process>[^\[:]+)(?:\[(?P<pid>\d+)\])?:\s*(?P<message>.*)$`,
	)
)

// Syslog severity levels (from RFC 5424, used in priority calculation).
var syslogSeverities = []string{
	"EMERGENCY", // 0
	"ALERT",     // 1
	"CRITICAL",  // 2
	"ERROR",     // 3
	"WARNING",   // 4
	"NOTICE",    // 5
	"INFO",      // 6
	"DEBUG",     // 7
}

// Parse reads syslog lines and emits LogEntry values.
func (p *SyslogParser) Parse(reader io.Reader) (<-chan LogEntry, <-chan error) {
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

			// Try with priority first, then without.
			matched := false
			for _, pattern := range []*regexp.Regexp{syslogWithPriority, syslogWithoutPriority} {
				match := pattern.FindStringSubmatch(raw)
				if match == nil {
					continue
				}

				for i, name := range pattern.SubexpNames() {
					if i == 0 || name == "" || match[i] == "" {
						continue
					}
					switch name {
					case "priority":
						pri, _ := strconv.Atoi(match[i])
						severity := pri & 0x07 // lower 3 bits
						facility := pri >> 3   // upper bits
						if severity < len(syslogSeverities) {
							entry.Level = NormalizeLevel(syslogSeverities[severity])
						}
						entry.Fields["priority"] = match[i]
						entry.Fields["facility"] = strconv.Itoa(facility)
						entry.Fields["severity"] = strconv.Itoa(severity)
					case "timestamp":
						entry.Timestamp = parseSyslogTimestamp(match[i])
					case "hostname":
						entry.Source = match[i]
						entry.Fields["hostname"] = match[i]
					case "process":
						entry.Fields["process"] = strings.TrimSpace(match[i])
					case "pid":
						entry.Fields["pid"] = match[i]
					case "message":
						entry.Message = match[i]
					}
				}
				matched = true
				break
			}

			if !matched {
				entry.Message = raw
				errs <- fmt.Errorf("line %d: does not match syslog format", lineNum)
			}

			entries <- entry
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("reading input: %w", err)
		}
	}()

	return entries, errs
}

// parseSyslogTimestamp parses the BSD syslog timestamp format (Mon DD HH:MM:SS).
// Since syslog timestamps don't include the year, we use the current year.
func parseSyslogTimestamp(s string) time.Time {
	// Normalize double-space after month (e.g., "Jun  5" vs "Jun 15").
	s = strings.Join(strings.Fields(s), " ")
	t, err := time.Parse("Jan 2 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	// Syslog doesn't include year; assume current year.
	now := time.Now()
	return t.AddDate(now.Year(), 0, 0)
}
