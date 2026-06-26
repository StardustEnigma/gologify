package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExport_JSONOutput(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.json")

	outFile := filepath.Join(t.TempDir(), "out.json")
	exportOutput = outFile
	exportFormat = "json"

	err := runExport(exportCmd, []string{inFile})
	if err != nil {
		t.Fatalf("export to JSON failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("exported file is empty")
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 10 {
		t.Errorf("expected at least 10 JSON lines, got %d", len(lines))
	}
}

func TestExport_CSVOutput(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.log")

	outFile := filepath.Join(t.TempDir(), "out.csv")
	exportOutput = outFile
	exportFormat = "csv"

	err := runExport(exportCmd, []string{inFile})
	if err != nil {
		t.Fatalf("export to CSV failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "line,timestamp,level,source,message") {
		t.Error("expected CSV header row")
	}
}

func TestExport_WithLevelFilter(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.json")

	outFile := filepath.Join(t.TempDir(), "errors.json")
	exportOutput = outFile
	exportFormat = "json"
	exportLevel = "ERROR"

	err := runExport(exportCmd, []string{inFile})
	if err != nil {
		t.Fatalf("export with level filter failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(line, `"level":"ERROR"`) {
			t.Errorf("expected all entries to be ERROR level, got: %s", line)
		}
	}
}

func TestExport_WithLimit(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.json")

	outFile := filepath.Join(t.TempDir(), "limited.json")
	exportOutput = outFile
	exportFormat = "json"
	exportLimit = 3

	err := runExport(exportCmd, []string{inFile})
	if err != nil {
		t.Fatalf("export with limit failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 3 {
		t.Errorf("expected 3 entries with --limit 3, got %d", nonEmpty)
	}
}

func TestExport_RawFormat(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.log")

	outFile := filepath.Join(t.TempDir(), "raw.log")
	exportOutput = outFile
	exportFormat = "raw"

	err := runExport(exportCmd, []string{inFile})
	if err != nil {
		t.Fatalf("export as raw failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("exported raw file is empty")
	}
}

func TestExport_EmptyOutputPath(t *testing.T) {
	resetAllFlags()
	dir := testdataDir(t)
	inFile := filepath.Join(dir, "sample.log")

	exportOutput = ""
	exportFormat = "json"

	err := runExport(exportCmd, []string{inFile})
	if err == nil {
		t.Error("expected error when output path is empty")
	}
}
