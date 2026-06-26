package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestStats_JSONLog(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	statsOutput = "json"

	getOutput := captureStdout(t)
	err := runStats(statsCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON stats output: %v\nOutput: %s", err, output)
	}

	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("missing 'summary' in stats output")
	}
	if summary["matched_entries"].(float64) == 0 {
		t.Error("expected matched_entries > 0")
	}
}

func TestStats_TextLog(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	getOutput := captureStdout(t)
	err := runStats(statsCmd, []string{file})
	_ = getOutput()

	if err != nil {
		t.Fatalf("stats on text log failed: %v", err)
	}
}

func TestStats_WithLevelFilter(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.json")

	statsLevel = "ERROR"
	statsOutput = "json"

	getOutput := captureStdout(t)
	err := runStats(statsCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("stats with level filter failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	summary := result["summary"].(map[string]interface{})
	matched := int(summary["matched_entries"].(float64))
	total := int(summary["total_entries"].(float64))
	if matched >= total {
		t.Errorf("with level=ERROR filter, matched (%d) should be less than total (%d)", matched, total)
	}
}

func TestStats_CSVOutput(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	statsOutput = "csv"

	getOutput := captureStdout(t)
	err := runStats(statsCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("stats CSV output failed: %v", err)
	}

	if !strings.Contains(output, "level,count") {
		t.Error("expected CSV header 'level,count'")
	}
}
