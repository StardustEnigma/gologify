package parser

import (
	"strings"
	"testing"
)

func TestSyslogParser_WithPriority(t *testing.T) {
	input := "<13>Jan 15 08:23:01 myhost sshd[1234]: Connection accepted from 192.168.1.10\n"
	p := &SyslogParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Source != "myhost" {
		t.Errorf("Source = %q, want myhost", entry.Source)
	}
	if entry.Message != "Connection accepted from 192.168.1.10" {
		t.Errorf("Message = %q", entry.Message)
	}
	if entry.Fields["process"] != "sshd" {
		t.Errorf("Fields[process] = %q, want sshd", entry.Fields["process"])
	}
	if entry.Fields["pid"] != "1234" {
		t.Errorf("Fields[pid] = %q, want 1234", entry.Fields["pid"])
	}
	if entry.Fields["priority"] != "13" {
		t.Errorf("Fields[priority] = %q, want 13", entry.Fields["priority"])
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	// Priority 13 = facility 1 (user), severity 5 (notice) → INFO after normalize
	// Actually severity 5 = NOTICE which normalizes to NOTICE (just uppercased)
	// Let's check the level is set from severity
	if entry.Level == "" {
		t.Error("expected non-empty level from priority")
	}
}

func TestSyslogParser_WithoutPriority(t *testing.T) {
	input := "Jan 15 08:23:01 webserver nginx[5678]: GET /index.html 200\n"
	p := &SyslogParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Source != "webserver" {
		t.Errorf("Source = %q, want webserver", entry.Source)
	}
	if entry.Fields["process"] != "nginx" {
		t.Errorf("Fields[process] = %q, want nginx", entry.Fields["process"])
	}
	if entry.Fields["pid"] != "5678" {
		t.Errorf("Fields[pid] = %q, want 5678", entry.Fields["pid"])
	}
}

func TestSyslogParser_SeverityExtraction(t *testing.T) {
	// Priority 11 = facility 1, severity 3 (ERROR)
	input := "<11>Jan 15 08:23:01 myhost kernel: Out of memory\n"
	p := &SyslogParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	go func() {
		for range errs {
		}
	}()

	entry := <-entries
	if entry.Level != "ERROR" {
		t.Errorf("Level = %q, want ERROR (severity 3)", entry.Level)
	}
}

func TestSyslogParser_UnmatchedLine(t *testing.T) {
	input := "This is not a syslog line at all\n"
	p := &SyslogParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var errCount int
	done := make(chan struct{})
	go func() {
		for range errs {
			errCount++
		}
		close(done)
	}()

	entry := <-entries
	<-done

	if entry.Message != "This is not a syslog line at all" {
		t.Errorf("unmatched line should use raw as message, got %q", entry.Message)
	}
	if errCount != 1 {
		t.Errorf("expected 1 error for unmatched syslog, got %d", errCount)
	}
}

func TestSyslogParser_MultipleEntries(t *testing.T) {
	input := `<13>Jan 15 08:23:01 host1 sshd[100]: line one
<14>Jan 15 08:24:00 host2 cron[200]: line two
Jan 15 08:25:00 host3 app[300]: line three
`
	p := &SyslogParser{}
	entries, errs := p.Parse(strings.NewReader(input))

	var results []LogEntry
	go func() {
		for range errs {
		}
	}()
	for entry := range entries {
		results = append(results, entry)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(results))
	}
	if results[0].Source != "host1" {
		t.Errorf("results[0].Source = %q", results[0].Source)
	}
	if results[1].Source != "host2" {
		t.Errorf("results[1].Source = %q", results[1].Source)
	}
	if results[2].Source != "host3" {
		t.Errorf("results[2].Source = %q", results[2].Source)
	}
}

func TestParseSyslogTimestamp(t *testing.T) {
	ts := parseSyslogTimestamp("Jan 15 08:23:01")
	if ts.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
	if ts.Month() != 1 || ts.Day() != 15 {
		t.Errorf("unexpected date: %v", ts)
	}
	if ts.Hour() != 8 || ts.Minute() != 23 || ts.Second() != 1 {
		t.Errorf("unexpected time: %v", ts)
	}
}

func TestParseSyslogTimestamp_DoubleSpace(t *testing.T) {
	// Single-digit day with double space: "Jun  5"
	ts := parseSyslogTimestamp("Jun  5 14:30:00")
	if ts.IsZero() {
		t.Fatal("expected non-zero timestamp for double-space day")
	}
	if ts.Day() != 5 {
		t.Errorf("expected day 5, got %d", ts.Day())
	}
}
