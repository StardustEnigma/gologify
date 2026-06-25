package parser

import (
	"strings"
	"testing"
)

func TestCSVParser_BasicParsing(t *testing.T) {
	input := `timestamp,level,message,source
2024-01-15T08:23:01Z,INFO,Application started,api-gateway
2024-01-15T08:24:01Z,ERROR,Connection failed,cache
`
	p := &CSVParser{}
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

	e := results[0]
	if e.Level != "INFO" {
		t.Errorf("entry[0].Level = %q, want INFO", e.Level)
	}
	if e.Message != "Application started" {
		t.Errorf("entry[0].Message = %q", e.Message)
	}
	if e.Source != "api-gateway" {
		t.Errorf("entry[0].Source = %q, want api-gateway", e.Source)
	}
	if e.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if e.LineNum != 2 { // header is line 1
		t.Errorf("entry[0].LineNum = %d, want 2", e.LineNum)
	}

	e2 := results[1]
	if e2.Level != "ERROR" {
		t.Errorf("entry[1].Level = %q, want ERROR", e2.Level)
	}
}

func TestCSVParser_AllFieldsInMap(t *testing.T) {
	input := `timestamp,level,message,status,duration
2024-01-15T08:23:01Z,INFO,Request handled,200,45
`
	p := &CSVParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Fields["status"] != "200" {
		t.Errorf("Fields[status] = %q, want 200", entry.Fields["status"])
	}
	if entry.Fields["duration"] != "45" {
		t.Errorf("Fields[duration] = %q, want 45", entry.Fields["duration"])
	}
}

func TestCSVParser_AlternateColumnNames(t *testing.T) {
	input := `ts,severity,msg,host
2024-01-15T08:23:01Z,WARN,High memory,server1
`
	p := &CSVParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Timestamp.IsZero() {
		t.Error("expected timestamp from 'ts' column")
	}
	if entry.Level != "WARN" {
		t.Errorf("Level = %q, want WARN (from severity column)", entry.Level)
	}
	if entry.Message != "High memory" {
		t.Errorf("Message = %q (from msg column)", entry.Message)
	}
	if entry.Source != "server1" {
		t.Errorf("Source = %q, want server1 (from host column)", entry.Source)
	}
}

func TestCSVParser_EmptyHeader(t *testing.T) {
	input := ""
	p := &CSVParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var errCount int
	done := make(chan struct{})
	go func() {
		for range errs {
			errCount++
		}
		close(done)
	}()
	for range entries {
	}
	<-done

	if errCount != 1 {
		t.Errorf("expected 1 error for empty CSV, got %d", errCount)
	}
}

func TestFindColumnIndex(t *testing.T) {
	headers := []string{"time", "level", "message", "source"}

	tests := []struct {
		candidates []string
		want       int
	}{
		{[]string{"timestamp", "time", "ts"}, 0},
		{[]string{"level", "severity"}, 1},
		{[]string{"message", "msg"}, 2},
		{[]string{"nonexistent"}, -1},
	}

	for _, tt := range tests {
		got := findColumnIndex(headers, tt.candidates...)
		if got != tt.want {
			t.Errorf("findColumnIndex(%v) = %d, want %d", tt.candidates, got, tt.want)
		}
	}
}
