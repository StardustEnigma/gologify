package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTail_DefaultLines(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	getOutput := captureStdout(t)
	err := runTail(tailCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("tail failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines (default), got %d", len(lines))
	}
}

func TestTail_CustomLineCount(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	tailLines = 5

	getOutput := captureStdout(t)
	err := runTail(tailCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("tail -n 5 failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

func TestTail_SearchFilter(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	file := filepath.Join(dir, "sample.log")

	tailLines = 25
	tailSearch = "ERROR"

	getOutput := captureStdout(t)
	err := runTail(tailCmd, []string{file})
	output := getOutput()

	if err != nil {
		t.Fatalf("tail with search failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !strings.Contains(strings.ToUpper(line), "ERROR") {
			t.Errorf("expected all lines to contain ERROR, got: %s", line)
		}
	}
}

func TestTail_NonexistentFile(t *testing.T) {
	resetAllFlags()
	err := runTail(tailCmd, []string{"nonexistent_file.log"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
