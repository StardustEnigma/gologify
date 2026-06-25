// Package filter provides composable log entry filters for searching,
// pattern matching, and field-based filtering.
package filter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/StardustEnigma/gologify/pkg/parser"
)

// Filter defines the interface for matching log entries.
type Filter interface {
	Match(entry parser.LogEntry) bool
}

// Chain combines multiple filters with AND logic.
// An entry must match all filters to pass through.
type Chain struct {
	filters []Filter
}

// NewChain creates a new filter chain from the given filters.
func NewChain(filters ...Filter) *Chain {
	return &Chain{filters: filters}
}

// Add appends a filter to the chain.
func (c *Chain) Add(f Filter) {
	c.filters = append(c.filters, f)
}

// Match returns true if the entry matches all filters in the chain.
// An empty chain matches everything.
func (c *Chain) Match(entry parser.LogEntry) bool {
	for _, f := range c.filters {
		if !f.Match(entry) {
			return false
		}
	}
	return true
}

// IsEmpty returns true if the chain has no filters.
func (c *Chain) IsEmpty() bool {
	return len(c.filters) == 0
}

// KeywordFilter matches entries containing a substring in the message,
// raw line, or any field value. Case-insensitive.
type KeywordFilter struct {
	keyword string
}

// NewKeywordFilter creates a filter that matches entries containing the keyword.
func NewKeywordFilter(keyword string) *KeywordFilter {
	return &KeywordFilter{keyword: strings.ToLower(keyword)}
}

// Match returns true if any searchable field contains the keyword.
func (f *KeywordFilter) Match(entry parser.LogEntry) bool {
	if containsLower(entry.Raw, f.keyword) {
		return true
	}
	if containsLower(entry.Message, f.keyword) {
		return true
	}
	if containsLower(entry.Level, f.keyword) {
		return true
	}
	if containsLower(entry.Source, f.keyword) {
		return true
	}
	for _, v := range entry.Fields {
		if containsLower(v, f.keyword) {
			return true
		}
	}
	return false
}

// FieldFilter matches entries where a specific field matches a pattern.
// Supports exact match, substring match, and regex patterns.
// Format: "field:pattern" where pattern can be a regex.
type FieldFilter struct {
	field   string
	pattern *regexp.Regexp
	raw     string
}

// NewFieldFilter creates a filter from a "field:pattern" expression.
func NewFieldFilter(expr string) (*FieldFilter, error) {
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filter expression %q (expected field:pattern)", expr)
	}

	field := strings.TrimSpace(parts[0])
	pattern := strings.TrimSpace(parts[1])

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex in filter %q: %w", expr, err)
	}

	return &FieldFilter{
		field:   strings.ToLower(field),
		pattern: re,
		raw:     pattern,
	}, nil
}

// Match returns true if the entry's field value matches the pattern.
func (f *FieldFilter) Match(entry parser.LogEntry) bool {
	// Check well-known fields first.
	switch f.field {
	case "level", "severity":
		return f.pattern.MatchString(entry.Level)
	case "message", "msg":
		return f.pattern.MatchString(entry.Message)
	case "source", "service", "host", "hostname":
		return f.pattern.MatchString(entry.Source)
	}

	// Check arbitrary fields (case-insensitive key lookup).
	for k, v := range entry.Fields {
		if strings.ToLower(k) == f.field {
			return f.pattern.MatchString(v)
		}
	}

	return false
}

// LevelFilter matches entries at or above a minimum log level.
type LevelFilter struct {
	minLevel int
	exact    bool
	target   string
}

// levelPriority maps normalized levels to numeric priorities.
var levelPriority = map[string]int{
	"TRACE": 0,
	"DEBUG": 1,
	"INFO":  2,
	"WARN":  3,
	"ERROR": 4,
	"FATAL": 5,
	"PANIC": 6,
}

// NewLevelFilter creates a filter for log level. If exact is true,
// only that specific level matches. Otherwise, that level and above match.
func NewLevelFilter(level string, exact bool) *LevelFilter {
	normalized := parser.NormalizeLevel(level)
	priority, ok := levelPriority[normalized]
	if !ok {
		priority = 2 // default to INFO
	}
	return &LevelFilter{
		minLevel: priority,
		exact:    exact,
		target:   normalized,
	}
}

// Match returns true if the entry meets the level criteria.
func (f *LevelFilter) Match(entry parser.LogEntry) bool {
	if entry.Level == "" {
		return false
	}
	if f.exact {
		return entry.Level == f.target
	}
	priority, ok := levelPriority[entry.Level]
	if !ok {
		return false
	}
	return priority >= f.minLevel
}

// RegexFilter matches entries whose raw line matches a regex pattern.
type RegexFilter struct {
	pattern *regexp.Regexp
}

// NewRegexFilter creates a filter from a regex pattern.
func NewRegexFilter(pattern string) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	return &RegexFilter{pattern: re}, nil
}

// Match returns true if the raw line matches the regex.
func (f *RegexFilter) Match(entry parser.LogEntry) bool {
	return f.pattern.MatchString(entry.Raw)
}

// TimeRangeFilter matches entries within a time range.
type TimeRangeFilter struct {
	from time.Time
	to   time.Time
}

// NewTimeRangeFilter creates a filter for a time range.
// Either from or to can be zero to leave that bound open.
func NewTimeRangeFilter(from, to time.Time) *TimeRangeFilter {
	return &TimeRangeFilter{from: from, to: to}
}

// Match returns true if the entry's timestamp is within the range.
func (f *TimeRangeFilter) Match(entry parser.LogEntry) bool {
	if entry.Timestamp.IsZero() {
		return false
	}
	if !f.from.IsZero() && entry.Timestamp.Before(f.from) {
		return false
	}
	if !f.to.IsZero() && entry.Timestamp.After(f.to) {
		return false
	}
	return true
}

// containsLower checks if s contains substr (case-insensitive).
func containsLower(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), substr)
}

// BuildChain creates a filter chain from command-line flag values.
// This is the bridge between CLI flags and the filter engine.
func BuildChain(search string, filters []string, level string, regex string, from string, to string) (*Chain, error) {
	chain := NewChain()

	if search != "" {
		chain.Add(NewKeywordFilter(search))
	}

	for _, expr := range filters {
		f, err := NewFieldFilter(expr)
		if err != nil {
			return nil, err
		}
		chain.Add(f)
	}

	if level != "" {
		chain.Add(NewLevelFilter(level, true))
	}

	if regex != "" {
		f, err := NewRegexFilter(regex)
		if err != nil {
			return nil, err
		}
		chain.Add(f)
	}

	if from != "" || to != "" {
		var fromTime, toTime time.Time
		var err error
		if from != "" {
			fromTime, err = time.Parse(time.RFC3339, from)
			if err != nil {
				return nil, fmt.Errorf("invalid --from time %q (expected RFC3339): %w", from, err)
			}
		}
		if to != "" {
			toTime, err = time.Parse(time.RFC3339, to)
			if err != nil {
				return nil, fmt.Errorf("invalid --to time %q (expected RFC3339): %w", to, err)
			}
		}
		chain.Add(NewTimeRangeFilter(fromTime, toTime))
	}

	return chain, nil
}
