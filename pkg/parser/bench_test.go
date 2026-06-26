package parser

import (
	"fmt"
	"strings"
	"testing"
)

// generateJSONLines creates n JSON log lines for benchmarking.
func generateJSONLines(n int) string {
	var b strings.Builder
	levels := []string{"info", "warn", "error", "debug"}
	for i := 0; i < n; i++ {
		level := levels[i%len(levels)]
		fmt.Fprintf(&b, `{"timestamp":"2024-01-15T08:%02d:%02dZ","level":"%s","msg":"Request processed %d","service":"api","status":%d,"duration_ms":%d}`+"\n",
			i/60%60, i%60, level, i, 200+(i%5)*100, i%1000)
	}
	return b.String()
}

// generateTextLines creates n plain text log lines for benchmarking.
func generateTextLines(n int) string {
	var b strings.Builder
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	for i := 0; i < n; i++ {
		level := levels[i%len(levels)]
		fmt.Fprintf(&b, "2024-01-15 08:%02d:%02d %s Request %d processed in %dms\n",
			i/60%60, i%60, level, i, i%1000)
	}
	return b.String()
}

// generateCSVLines creates n CSV log lines (plus header) for benchmarking.
func generateCSVLines(n int) string {
	var b strings.Builder
	b.WriteString("timestamp,level,source,message,status,duration_ms\n")
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	for i := 0; i < n; i++ {
		level := levels[i%len(levels)]
		fmt.Fprintf(&b, "2024-01-15T08:%02d:%02dZ,%s,api,Request %d processed,%d,%d\n",
			i/60%60, i%60, level, i, 200+(i%5)*100, i%1000)
	}
	return b.String()
}

// generateSyslogLines creates n syslog lines for benchmarking.
func generateSyslogLines(n int) string {
	var b strings.Builder
	processes := []string{"api-gateway", "scheduler", "db-proxy", "cache"}
	for i := 0; i < n; i++ {
		proc := processes[i%len(processes)]
		fmt.Fprintf(&b, "Jan 15 08:%02d:%02d web-server-01 %s[%d]: Request %d processed\n",
			i/60%60, i%60, proc, 1000+i%100, i)
	}
	return b.String()
}

// drainParser consumes all entries and errors from a parser.
func drainParser(p Parser, input string) {
	reader := strings.NewReader(input)
	entries, errs := p.Parse(reader)

	go func() {
		for range errs {
		}
	}()

	for range entries {
	}
}

func BenchmarkJSONParser_1K(b *testing.B) {
	input := generateJSONLines(1000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&JSONParser{}, input)
	}
}

func BenchmarkJSONParser_10K(b *testing.B) {
	input := generateJSONLines(10000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&JSONParser{}, input)
	}
}

func BenchmarkTextParser_1K(b *testing.B) {
	input := generateTextLines(1000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&TextParser{}, input)
	}
}

func BenchmarkTextParser_10K(b *testing.B) {
	input := generateTextLines(10000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&TextParser{}, input)
	}
}

func BenchmarkCSVParser_1K(b *testing.B) {
	input := generateCSVLines(1000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&CSVParser{}, input)
	}
}

func BenchmarkCSVParser_10K(b *testing.B) {
	input := generateCSVLines(10000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&CSVParser{}, input)
	}
}

func BenchmarkSyslogParser_1K(b *testing.B) {
	input := generateSyslogLines(1000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&SyslogParser{}, input)
	}
}

func BenchmarkSyslogParser_10K(b *testing.B) {
	input := generateSyslogLines(10000)
	b.ResetTimer()
	for b.Loop() {
		drainParser(&SyslogParser{}, input)
	}
}

func BenchmarkConcurrentParser_JSON_10K(b *testing.B) {
	input := generateJSONLines(10000)
	b.ResetTimer()
	for b.Loop() {
		p := NewConcurrentParser(FormatJSON, 4)
		reader := strings.NewReader(input)
		entries, errs := p.Parse(reader)
		go func() { for range errs {} }()
		for range entries {}
	}
}

func BenchmarkFormatDetection(b *testing.B) {
	input := generateJSONLines(100)
	b.ResetTimer()
	for b.Loop() {
		reader := strings.NewReader(input)
		DetectFormat(reader)
	}
}
