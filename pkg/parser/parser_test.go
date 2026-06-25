package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"auto", FormatAuto, false},
		{"", FormatAuto, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"text", FormatText, false},
		{"plain", FormatText, false},
		{"txt", FormatText, false},
		{"csv", FormatCSV, false},
		{"syslog", FormatSyslog, false},
		{"  json  ", FormatJSON, false},
		{"xml", "", true},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"info", "INFO"},
		{"INFO", "INFO"},
		{"INF", "INFO"},
		{"INFORMATION", "INFO"},
		{"warn", "WARN"},
		{"WRN", "WARN"},
		{"WARNING", "WARN"},
		{"error", "ERROR"},
		{"ERR", "ERROR"},
		{"debug", "DEBUG"},
		{"DBG", "DEBUG"},
		{"DEBU", "DEBUG"},
		{"fatal", "FATAL"},
		{"FTL", "FATAL"},
		{"CRITICAL", "FATAL"},
		{"CRIT", "FATAL"},
		{"panic", "PANIC"},
		{"TRACE", "TRACE"},
		{"TRC", "TRACE"},
		{"CUSTOM", "CUSTOM"},
		{"  info  ", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeLevel(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Format
	}{
		{
			name:  "JSON lines",
			input: "{\"level\":\"info\",\"msg\":\"hello\"}\n{\"level\":\"error\",\"msg\":\"fail\"}\n",
			want:  FormatJSON,
		},
		{
			name:  "CSV data",
			input: "timestamp,level,message\n2024-01-01,INFO,hello\n2024-01-02,ERROR,fail\n",
			want:  FormatCSV,
		},
		{
			name:  "syslog with priority",
			input: "<13>Jan 15 08:23:01 myhost sshd[1234]: Connection accepted\n<11>Jan 15 08:24:00 myhost kernel: Out of memory\n",
			want:  FormatSyslog,
		},
		{
			name:  "syslog without priority",
			input: "Jan 15 08:23:01 myhost sshd[1234]: Connection accepted\nJan 15 08:24:00 myhost kernel: Out of memory\n",
			want:  FormatSyslog,
		},
		{
			name:  "plain text",
			input: "2024-01-15 08:23:01 INFO Application started\n2024-01-15 08:23:02 ERROR Something failed\n",
			want:  FormatText,
		},
		{
			name:  "empty input",
			input: "",
			want:  FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			got, _ := DetectFormat(reader)
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryParseTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2024-01-15T08:23:01Z", true},
		{"2024-01-15T08:23:01.123Z", true},
		{"2024-01-15T08:23:01", true},
		{"2024-01-15 08:23:01", true},
		{"2024-01-15 08:23:01.000", true},
		{"2024-01-15 08:23:01,000", true},
		{"2024/01/02 15:04:05", true},
		{"not a timestamp", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, ok := tryParseTimestamp(tt.input)
			if ok != tt.want {
				t.Errorf("tryParseTimestamp(%q) ok = %v, want %v", tt.input, ok, tt.want)
			}
		})
	}
}

func TestNewParser(t *testing.T) {
	tests := []struct {
		format Format
		typ    string
	}{
		{FormatJSON, "*parser.JSONParser"},
		{FormatCSV, "*parser.CSVParser"},
		{FormatSyslog, "*parser.SyslogParser"},
		{FormatText, "*parser.TextParser"},
		{FormatAuto, "*parser.TextParser"}, // default fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			p := NewParser(tt.format)
			if p == nil {
				t.Fatal("NewParser returned nil")
			}
		})
	}
}

func TestTryParseTimestampValues(t *testing.T) {
	ts, ok := tryParseTimestamp("2024-01-15T08:23:01Z")
	if !ok {
		t.Fatal("expected successful parse")
	}
	if ts.Year() != 2024 || ts.Month() != time.January || ts.Day() != 15 {
		t.Errorf("unexpected date: %v", ts)
	}
	if ts.Hour() != 8 || ts.Minute() != 23 || ts.Second() != 1 {
		t.Errorf("unexpected time: %v", ts)
	}
}
