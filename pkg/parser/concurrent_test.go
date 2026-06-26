package parser

import (
	"strings"
	"testing"
)

func TestConcurrentParser_JSON_MatchesSequential(t *testing.T) {
	input := generateJSONLines(100)

	// Parse sequentially.
	seqParser := NewParser(FormatJSON)
	seqEntries, seqErrs := seqParser.Parse(strings.NewReader(input))
	go func() { for range seqErrs {} }()
	var seqResults []LogEntry
	for e := range seqEntries {
		seqResults = append(seqResults, e)
	}

	// Parse concurrently with 4 workers.
	concParser := NewConcurrentParser(FormatJSON, 4)
	concEntries, concErrs := concParser.Parse(strings.NewReader(input))
	go func() { for range concErrs {} }()
	var concResults []LogEntry
	for e := range concEntries {
		concResults = append(concResults, e)
	}

	if len(seqResults) != len(concResults) {
		t.Fatalf("entry count mismatch: sequential=%d, concurrent=%d", len(seqResults), len(concResults))
	}

	for i := range seqResults {
		if seqResults[i].LineNum != concResults[i].LineNum {
			t.Errorf("entry %d: line number mismatch: seq=%d, conc=%d", i, seqResults[i].LineNum, concResults[i].LineNum)
		}
		if seqResults[i].Message != concResults[i].Message {
			t.Errorf("entry %d: message mismatch: seq=%q, conc=%q", i, seqResults[i].Message, concResults[i].Message)
		}
		if seqResults[i].Level != concResults[i].Level {
			t.Errorf("entry %d: level mismatch: seq=%q, conc=%q", i, seqResults[i].Level, concResults[i].Level)
		}
	}
}

func TestConcurrentParser_Text_MatchesSequential(t *testing.T) {
	input := generateTextLines(100)

	seqParser := NewParser(FormatText)
	seqEntries, seqErrs := seqParser.Parse(strings.NewReader(input))
	go func() { for range seqErrs {} }()
	var seqResults []LogEntry
	for e := range seqEntries {
		seqResults = append(seqResults, e)
	}

	concParser := NewConcurrentParser(FormatText, 4)
	concEntries, concErrs := concParser.Parse(strings.NewReader(input))
	go func() { for range concErrs {} }()
	var concResults []LogEntry
	for e := range concEntries {
		concResults = append(concResults, e)
	}

	if len(seqResults) != len(concResults) {
		t.Fatalf("entry count mismatch: sequential=%d, concurrent=%d", len(seqResults), len(concResults))
	}

	for i := range seqResults {
		if seqResults[i].LineNum != concResults[i].LineNum {
			t.Errorf("entry %d: line number mismatch: seq=%d, conc=%d", i, seqResults[i].LineNum, concResults[i].LineNum)
		}
	}
}

func TestConcurrentParser_SingleWorker_FallsBack(t *testing.T) {
	// workers=1 should return a non-concurrent parser.
	p := NewConcurrentParser(FormatJSON, 1)
	if _, ok := p.(*ConcurrentParser); ok {
		t.Error("workers=1 should return a sequential parser, not ConcurrentParser")
	}
}

func TestConcurrentParser_ZeroWorkers_AutoDetects(t *testing.T) {
	// workers=0 should auto-detect and return ConcurrentParser (since NumCPU >= 1 on any real system).
	p := NewConcurrentParser(FormatJSON, 0)
	// On single-core, this may be sequential; on multi-core, concurrent.
	// We just verify it doesn't panic and parses correctly.
	input := generateJSONLines(10)
	entries, errs := p.Parse(strings.NewReader(input))
	go func() { for range errs {} }()
	count := 0
	for range entries {
		count++
	}
	if count != 10 {
		t.Errorf("expected 10 entries, got %d", count)
	}
}

func TestConcurrentParser_EmptyInput(t *testing.T) {
	p := NewConcurrentParser(FormatJSON, 4)
	entries, errs := p.Parse(strings.NewReader(""))
	go func() { for range errs {} }()
	count := 0
	for range entries {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", count)
	}
}
