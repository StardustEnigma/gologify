package parser

import (
	"strings"
	"testing"
)

func TestTextParser_TimestampLevelMessage(t *testing.T) {
	input := "2024-01-15 08:23:01 INFO Application started successfully\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Message != "Application started successfully" {
		t.Errorf("Message = %q", entry.Message)
	}
}

func TestTextParser_BracketedFormat(t *testing.T) {
	input := "[2024-01-15T10:30:45Z] [ERROR] Something went wrong\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Level != "ERROR" {
		t.Errorf("Level = %q, want ERROR", entry.Level)
	}
	if entry.Message != "Something went wrong" {
		t.Errorf("Message = %q", entry.Message)
	}
}

func TestTextParser_PythonStyle(t *testing.T) {
	input := "2024-01-15 10:30:45 - WARNING - Disk space low\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Level != "WARN" {
		t.Errorf("Level = %q, want WARN", entry.Level)
	}
	if entry.Message != "Disk space low" {
		t.Errorf("Message = %q", entry.Message)
	}
}

func TestTextParser_LevelFirst(t *testing.T) {
	input := "ERROR 2024-01-15 10:30:45 Database connection lost\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Level != "ERROR" {
		t.Errorf("Level = %q, want ERROR", entry.Level)
	}
}

func TestTextParser_TimestampOnly(t *testing.T) {
	input := "2024-01-15 10:30:45 Some message without a level\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if entry.Message != "Some message without a level" {
		t.Errorf("Message = %q", entry.Message)
	}
}

func TestTextParser_UnmatchedLine(t *testing.T) {
	input := "This is a completely unstructured line\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Message != "This is a completely unstructured line" {
		t.Errorf("unmatched line should use raw as message, got %q", entry.Message)
	}
	if entry.Level != "" {
		t.Errorf("unmatched line should have empty level, got %q", entry.Level)
	}
}

func TestTextParser_EmptyLines(t *testing.T) {
	input := "\n\n2024-01-15 08:23:01 INFO Hello\n\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var results []LogEntry
	go func() {
		for range errs {
		}
	}()
	for entry := range entries {
		results = append(results, entry)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 entry (empty lines skipped), got %d", len(results))
	}
}

func TestTextParser_WithSource(t *testing.T) {
	input := "2024-01-15 08:23:01 INFO [myapp] Request handled\n"
	p := &TextParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Source != "myapp" {
		t.Errorf("Source = %q, want myapp", entry.Source)
	}
}
