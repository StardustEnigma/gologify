package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/StardustEnigma/gologify/pkg/aggregator"
	"github.com/StardustEnigma/gologify/pkg/parser"
)

func makeEntry(level, message, source string, fields map[string]string, ts time.Time, lineNum int) parser.LogEntry {
	if fields == nil {
		fields = make(map[string]string)
	}
	return parser.LogEntry{
		Timestamp: ts,
		Level:     level,
		Message:   message,
		Source:    source,
		Fields:    fields,
		LineNum:   lineNum,
	}
}

// --- JSON Formatter ---

func TestJSONFormatter_FormatEntry(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	ts := time.Date(2024, 1, 15, 8, 23, 1, 0, time.UTC)
	entry := makeEntry("INFO", "Request processed", "api", nil, ts, 1)

	if err := f.FormatEntry(entry); err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if result["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", result["level"])
	}
	if result["message"] != "Request processed" {
		t.Errorf("message = %v", result["message"])
	}
	if result["source"] != "api" {
		t.Errorf("source = %v", result["source"])
	}
	if result["timestamp"] != "2024-01-15T08:23:01Z" {
		t.Errorf("timestamp = %v", result["timestamp"])
	}
}

func TestJSONFormatter_FormatEntries(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	entries := []parser.LogEntry{
		makeEntry("INFO", "msg1", "", nil, time.Time{}, 1),
		makeEntry("ERROR", "msg2", "", nil, time.Time{}, 2),
	}

	if err := f.FormatEntries(entries); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 JSON lines, got %d", len(lines))
	}
}

func TestJSONFormatter_OmitsZeroTimestamp(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	entry := makeEntry("INFO", "hello", "", nil, time.Time{}, 1)
	if err := f.FormatEntry(entry); err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)

	if _, exists := result["timestamp"]; exists {
		t.Error("expected timestamp to be omitted for zero time")
	}
}

func TestJSONFormatter_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	entry := makeEntry("INFO", "hello", "", nil, time.Time{}, 1)
	if err := f.FormatEntry(entry); err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)

	if _, exists := result["fields"]; exists {
		t.Error("expected fields to be omitted when empty")
	}
}

func TestJSONFormatter_FormatResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	result := aggregator.Result{
		TotalEntries:   100,
		MatchedEntries: 50,
		LevelCounts:    map[string]int{"INFO": 30, "ERROR": 20},
		FirstTimestamp: "2024-01-15T08:00:00Z",
		LastTimestamp:   "2024-01-15T09:00:00Z",
	}
	topErrs := []aggregator.TopEntry{
		{Value: "Connection failed", Count: 10},
	}

	if err := f.FormatResult(result, topErrs); err != nil {
		t.Fatal(err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}

	summary, ok := output["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("missing summary in output")
	}
	if summary["matched_entries"].(float64) != 50 {
		t.Errorf("matched_entries = %v", summary["matched_entries"])
	}
}

// --- CSV Formatter ---

func TestCSVFormatter_FormatEntries(t *testing.T) {
	var buf bytes.Buffer
	f := NewCSVFormatter(&buf)

	ts := time.Date(2024, 1, 15, 8, 23, 1, 0, time.UTC)
	entries := []parser.LogEntry{
		makeEntry("INFO", "hello", "api", map[string]string{"status": "200"}, ts, 1),
		makeEntry("ERROR", "fail", "db", map[string]string{"status": "500"}, ts, 2),
	}

	if err := f.FormatEntries(entries); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// Verify header includes the extra field.
	if !strings.Contains(lines[0], "status") {
		t.Error("header should contain 'status' field")
	}
}

func TestCSVFormatter_EmptyEntries(t *testing.T) {
	var buf bytes.Buffer
	f := NewCSVFormatter(&buf)

	if err := f.FormatEntries(nil); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Error("expected empty output for no entries")
	}
}

func TestCSVFormatter_FormatResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewCSVFormatter(&buf)

	result := aggregator.Result{
		LevelCounts: map[string]int{"INFO": 10, "ERROR": 5},
	}

	if err := f.FormatResult(result, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "level,count") {
		t.Error("expected CSV header for level counts")
	}
}

// --- Raw Formatter ---

func TestRawFormatter_NoHighlight(t *testing.T) {
	var buf bytes.Buffer
	f := NewRawFormatter(&buf, "")

	entry := parser.LogEntry{Raw: "2024-01-15 INFO Application started"}
	f.FormatEntry(entry)

	output := strings.TrimSpace(buf.String())
	if output != "2024-01-15 INFO Application started" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestRawFormatter_FormatEntries(t *testing.T) {
	var buf bytes.Buffer
	f := NewRawFormatter(&buf, "")

	entries := []parser.LogEntry{
		{Raw: "line 1"},
		{Raw: "line 2"},
	}
	f.FormatEntries(entries)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// --- Helper Functions ---

func TestHighlightText_NoPattern(t *testing.T) {
	result := HighlightText("hello world", "")
	if result != "hello world" {
		t.Errorf("expected unchanged text, got %q", result)
	}
}

func TestFormatLevelColor_ContainsLevel(t *testing.T) {
	// We can't easily test ANSI colors, but we can verify the function
	// returns a non-empty string and doesn't panic.
	levels := []string{"ERROR", "FATAL", "PANIC", "WARN", "WARNING", "INFO", "DEBUG"}
	for _, level := range levels {
		line := "2024-01-15 " + level + " some message"
		result := FormatLevelColor(line)
		if result == "" {
			t.Errorf("FormatLevelColor returned empty for %s", level)
		}
	}
}

func TestFormatLevelColor_NoLevel(t *testing.T) {
	line := "just a plain line"
	result := FormatLevelColor(line)
	if result != line {
		t.Errorf("expected unchanged line, got %q", result)
	}
}

// --- Table Formatter ---

func TestTableFormatter_FormatEntries_Empty(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf)
	f.FormatEntries(nil)

	if !strings.Contains(buf.String(), "No entries to display") {
		t.Error("expected 'No entries to display' message")
	}
}

func TestTableFormatter_FormatEntries_NonEmpty(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf)

	ts := time.Date(2024, 1, 15, 8, 23, 1, 0, time.UTC)
	entries := []parser.LogEntry{
		makeEntry("INFO", "hello world", "api", nil, ts, 1),
	}
	f.FormatEntries(entries)

	output := buf.String()
	if !strings.Contains(output, "hello world") {
		t.Error("expected table to contain message")
	}
}

func TestTableFormatter_FormatResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf)

	result := aggregator.Result{
		TotalEntries:   100,
		MatchedEntries: 50,
		LevelCounts:    map[string]int{"INFO": 30, "ERROR": 20},
		FirstTimestamp: "2024-01-15T08:00:00Z",
		LastTimestamp:   "2024-01-15T09:00:00Z",
	}
	topErrs := []aggregator.TopEntry{
		{Value: "Connection failed", Count: 10},
	}

	f.FormatResult(result, topErrs)

	output := buf.String()
	if !strings.Contains(output, "Log Analysis Summary") {
		t.Error("expected summary header")
	}
	if !strings.Contains(output, "Level Distribution") {
		t.Error("expected level distribution section")
	}
	if !strings.Contains(output, "Top Errors") {
		t.Error("expected top errors section")
	}
}

func TestNewTableFormatter_NilWriter(t *testing.T) {
	f := NewTableFormatter(nil)
	if f.writer == nil {
		t.Error("expected default writer (os.Stdout) for nil input")
	}
}

func TestCollectFieldKeys(t *testing.T) {
	entries := []parser.LogEntry{
		{Fields: map[string]string{"a": "1", "b": "2"}},
		{Fields: map[string]string{"b": "3", "c": "4"}},
	}

	keys := collectFieldKeys(entries)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	// Should be sorted.
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("keys = %v, want [a b c]", keys)
	}
}
