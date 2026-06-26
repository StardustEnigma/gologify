package output

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
)

// RawFormatter prints filtered log lines with optional keyword highlighting.
type RawFormatter struct {
	writer       io.Writer
	highlightRe  *regexp.Regexp
	highlightFn  func(string, ...interface{}) string
}

// NewRawFormatter creates a raw formatter. If highlight is non-empty,
// matching text will be color-highlighted in the output.
func NewRawFormatter(w io.Writer, highlight string) *RawFormatter {
	if w == nil {
		w = os.Stdout
	}

	f := &RawFormatter{
		writer:      w,
		highlightFn: color.New(color.BgYellow, color.FgBlack).Sprintf,
	}

	if highlight != "" {
		// Escape regex metacharacters for literal matching.
		escaped := regexp.QuoteMeta(highlight)
		f.highlightRe = regexp.MustCompile("(?i)" + escaped)
	}

	return f
}

// FormatEntry writes a single raw log line with optional highlighting.
func (f *RawFormatter) FormatEntry(entry parser.LogEntry) {
	line := entry.Raw
	if f.highlightRe != nil {
		line = f.highlightRe.ReplaceAllStringFunc(line, func(match string) string {
			return f.highlightFn(match)
		})
	}
	_, _ = fmt.Fprintln(f.writer, line)
}

// FormatEntries writes multiple raw log lines.
func (f *RawFormatter) FormatEntries(entries []parser.LogEntry) {
	for _, entry := range entries {
		f.FormatEntry(entry)
	}
}

// HighlightText applies highlighting to arbitrary text (for tail command).
func HighlightText(text string, pattern string) string {
	if pattern == "" {
		return text
	}
	escaped := regexp.QuoteMeta(pattern)
	re := regexp.MustCompile("(?i)" + escaped)
	highlighter := color.New(color.BgYellow, color.FgBlack).Sprintf

	return re.ReplaceAllStringFunc(text, func(match string) string {
		return highlighter(match)
	})
}

// FormatLevelColor applies color to log level text embedded in a line.
func FormatLevelColor(line string) string {
	replacements := []struct {
		level string
		fn    func(a ...interface{}) string
	}{
		{"ERROR", color.New(color.FgRed, color.Bold).Sprint},
		{"FATAL", color.New(color.FgRed, color.Bold).Sprint},
		{"PANIC", color.New(color.FgRed, color.Bold).Sprint},
		{"WARN", color.New(color.FgYellow).Sprint},
		{"WARNING", color.New(color.FgYellow).Sprint},
		{"INFO", color.New(color.FgGreen).Sprint},
		{"DEBUG", color.New(color.FgCyan).Sprint},
	}

	for _, r := range replacements {
		if strings.Contains(line, r.level) {
			line = strings.Replace(line, r.level, r.fn(r.level), 1)
			break
		}
	}
	return line
}
