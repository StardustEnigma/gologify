package parser

import (
	"strings"
	"testing"
)

func TestJSONParser_BasicParsing(t *testing.T) {
	input := `{"timestamp":"2024-01-15T08:23:01Z","level":"info","msg":"Application started","service":"api-gateway"}
{"timestamp":"2024-01-15T08:24:01Z","level":"error","msg":"Connection failed","host":"127.0.0.1","port":"6379"}
`
	p := &JSONParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var results []LogEntry
	go func() {
		for range errs {
		}
	}()
	for entry := range entries {
		results = append(results, entry)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(results))
	}

	// First entry.
	e := results[0]
	if e.Level != "INFO" {
		t.Errorf("entry[0].Level = %q, want %q", e.Level, "INFO")
	}
	if e.Message != "Application started" {
		t.Errorf("entry[0].Message = %q, want %q", e.Message, "Application started")
	}
	if e.Source != "api-gateway" {
		t.Errorf("entry[0].Source = %q, want %q", e.Source, "api-gateway")
	}
	if e.Timestamp.IsZero() {
		t.Error("entry[0].Timestamp should not be zero")
	}
	if e.LineNum != 1 {
		t.Errorf("entry[0].LineNum = %d, want 1", e.LineNum)
	}

	// Second entry.
	e2 := results[1]
	if e2.Level != "ERROR" {
		t.Errorf("entry[1].Level = %q, want %q", e2.Level, "ERROR")
	}
	if e2.Fields["port"] != "6379" {
		t.Errorf("entry[1].Fields[port] = %q, want %q", e2.Fields["port"], "6379")
	}
}

func TestJSONParser_MalformedLine(t *testing.T) {
	input := `not json at all
{"level":"info","msg":"valid line"}
`
	p := &JSONParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var results []LogEntry
	var errCount int
	done := make(chan struct{})
	go func() {
		for range errs {
			errCount++
		}
		close(done)
	}()
	for entry := range entries {
		results = append(results, entry)
	}
	<-done

	if len(results) != 2 {
		t.Fatalf("expected 2 entries (including malformed), got %d", len(results))
	}
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}

	// Malformed line should still be emitted as raw.
	if results[0].Raw != "not json at all" {
		t.Errorf("malformed entry.Raw = %q", results[0].Raw)
	}
}

func TestJSONParser_EmptyLines(t *testing.T) {
	input := `
{"level":"info","msg":"hello"}

{"level":"warn","msg":"world"}

`
	p := &JSONParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var results []LogEntry
	go func() {
		for range errs {
		}
	}()
	for entry := range entries {
		results = append(results, entry)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 entries (empty lines skipped), got %d", len(results))
	}
}

func TestJSONParser_EpochTimestamp(t *testing.T) {
	input := `{"ts":1705306981.5,"level":"info","msg":"epoch time"}
`
	p := &JSONParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp from epoch")
	}
	if entry.Timestamp.Year() != 2024 {
		t.Errorf("expected year 2024, got %d", entry.Timestamp.Year())
	}
}

func TestJSONParser_CaseInsensitiveFields(t *testing.T) {
	input := `{"Level":"WARN","Message":"case test","Timestamp":"2024-01-15T08:00:00Z"}
`
	p := &JSONParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Level != "WARN" {
		t.Errorf("Level = %q, want WARN (case-insensitive lookup)", entry.Level)
	}
	if entry.Message != "case test" {
		t.Errorf("Message = %q, want %q", entry.Message, "case test")
	}
}

func TestCaseInsensitiveLookup(t *testing.T) {
	fields := map[string]interface{}{
		"Level": "info",
		"Msg":   "hello",
	}

	v, ok := caseInsensitiveLookup(fields, "level")
	if !ok {
		t.Fatal("expected to find 'level' case-insensitively")
	}
	if v != "info" {
		t.Errorf("got %v, want 'info'", v)
	}

	_, ok = caseInsensitiveLookup(fields, "nonexistent")
	if ok {
		t.Error("expected false for nonexistent key")
	}
}
