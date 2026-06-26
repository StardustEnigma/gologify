package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testdataDir returns the path to the examples directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "examples"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("examples directory not found at %s", dir)
	}
	return dir
}

// captureStdout redirects os.Stdout to a pipe and returns a function
// that restores stdout and returns the captured output.
func captureStdout(t *testing.T) func() string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	// Drain the pipe in a goroutine to prevent deadlock when
	// output exceeds the OS pipe buffer size.
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		buf.ReadFrom(r)
		close(done)
	}()

	return func() string {
		w.Close()
		os.Stdout = old
		<-done
		return buf.String()
	}
}

// resetAllFlags resets all global flag variables to their defaults.
// This must be called before each test since cobra binds flags to globals.
func resetAllFlags() {
	// analyze flags
	analyzeFormat = "auto"
	analyzeLimit = 0
	searchTerm = ""
	filterExprs = nil
	levelFilter = ""
	regexFilter = ""
	timeFrom = ""
	timeTo = ""
	aggregate = false
	groupBy = ""
	topIPs = 0
	topErrors = 0
	outputFormat = "table"
	highlight = ""
	workers = 0

	// stats flags
	statsFormat = "auto"
	statsTopIPs = 10
	statsTopErrors = 10
	statsSearch = ""
	statsLevel = ""
	statsOutput = "table"

	// tail flags
	tailFollow = false
	tailLines = 10
	tailHighlight = ""
	tailFormat = "auto"
	tailSearch = ""
	tailLevel = ""
	tailColorize = true

	// export flags
	exportOutput = ""
	exportFormat = "json"
	exportLogFormat = "auto"
	exportSearch = ""
	exportFilter = nil
	exportLevel = ""
	exportRegex = ""
	exportFrom = ""
	exportTo = ""
	exportLimit = 0

	// global flags
	verbose = false
	noColor = false
}

// --- Analyze Command Tests ---

func TestAnalyze_TextLog(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if !strings.Contains(output, "Application started") {
		t.Error("expected output to contain 'Application started'")
	}
}

func TestAnalyze_JSONLog(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestAnalyze_SearchFilter(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	searchTerm = "error"
	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze with search failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one result for 'error' search")
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("invalid JSON line: %s", line)
		}
	}
}

func TestAnalyze_AggregateJSON(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	aggregate = true
	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze aggregate failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON aggregation output: %v\nOutput: %s", err, output)
	}

	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("missing 'summary' key in aggregation result")
	}
	if summary["matched_entries"].(float64) == 0 {
		t.Error("expected matched_entries > 0")
	}
}

func TestAnalyze_CSVLog(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.csv")

	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze CSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		t.Errorf("expected at least 5 entries from CSV, got %d", len(lines))
	}
}

func TestAnalyze_SyslogFile(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.syslog")

	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	_ = getOutput()

	if err != nil {
		t.Fatalf("analyze syslog failed: %v", err)
	}
}

func TestAnalyze_LevelFilter(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	levelFilter = "ERROR"
	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze with level filter failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("invalid JSON: %s", line)
			continue
		}
		if entry["level"] != "ERROR" {
			t.Errorf("expected level=ERROR, got %v", entry["level"])
		}
	}
}

func TestAnalyze_NonexistentFile(t *testing.T) {
	resetAllFlags()
	err := runAnalyze(analyzeCmd, []string{"nonexistent_file_that_does_not_exist.log"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestAnalyze_Limit(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	analyzeLimit = 3
	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze with limit failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty > 3 {
		t.Errorf("expected at most 3 entries with --limit 3, got %d", nonEmpty)
	}
}

func TestAnalyze_Workers(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	workers = 4
	aggregate = true
	outputFormat = "json"

	getOutput := captureStdout(t)
	err := runAnalyze(analyzeCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("analyze with workers failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON from concurrent parser: %v", err)
	}
}
