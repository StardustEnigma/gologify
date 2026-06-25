package filter

import (
	"testing"
	"time"

	"github.com/StardustEnigma/gologify/pkg/parser"
)

func makeEntry(level, message, source, raw string, fields map[string]string, ts time.Time) parser.LogEntry {
	if fields == nil {
		fields = make(map[string]string)
	}
	return parser.LogEntry{
		Timestamp: ts,
		Level:     level,
		Message:   message,
		Source:    source,
		Fields:    fields,
		Raw:       raw,
	}
}

// --- KeywordFilter ---

func TestKeywordFilter_MatchMessage(t *testing.T) {
	f := NewKeywordFilter("error")
	entry := makeEntry("INFO", "An error occurred", "", "An error occurred", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on message containing 'error'")
	}
}

func TestKeywordFilter_MatchLevel(t *testing.T) {
	f := NewKeywordFilter("error")
	entry := makeEntry("ERROR", "Something happened", "", "", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on level 'ERROR'")
	}
}

func TestKeywordFilter_MatchSource(t *testing.T) {
	f := NewKeywordFilter("gateway")
	entry := makeEntry("INFO", "hello", "api-gateway", "", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on source containing 'gateway'")
	}
}

func TestKeywordFilter_MatchFields(t *testing.T) {
	f := NewKeywordFilter("redis")
	entry := makeEntry("INFO", "hello", "", "", map[string]string{"service": "redis-cache"}, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on field value containing 'redis'")
	}
}

func TestKeywordFilter_MatchRaw(t *testing.T) {
	f := NewKeywordFilter("timeout")
	entry := makeEntry("", "", "", "connection timeout after 30s", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on raw line containing 'timeout'")
	}
}

func TestKeywordFilter_NoMatch(t *testing.T) {
	f := NewKeywordFilter("database")
	entry := makeEntry("INFO", "Request processed", "api", "Request processed", nil, time.Time{})
	if f.Match(entry) {
		t.Error("expected no match")
	}
}

func TestKeywordFilter_CaseInsensitive(t *testing.T) {
	f := NewKeywordFilter("ERROR")
	entry := makeEntry("", "An error occurred", "", "", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected case-insensitive match")
	}
}

// --- FieldFilter ---

func TestFieldFilter_LevelMatch(t *testing.T) {
	f, err := NewFieldFilter("level:ERROR")
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("ERROR", "fail", "", "", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on level field")
	}
}

func TestFieldFilter_MessageMatch(t *testing.T) {
	f, err := NewFieldFilter("message:timeout")
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "connection timeout", "", "", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on message field")
	}
}

func TestFieldFilter_CustomFieldMatch(t *testing.T) {
	f, err := NewFieldFilter("status:500")
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "", "", "", map[string]string{"status": "500"}, time.Time{})
	if !f.Match(entry) {
		t.Error("expected match on custom field")
	}
}

func TestFieldFilter_RegexPattern(t *testing.T) {
	f, err := NewFieldFilter("status:5\\d{2}")
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "", "", "", map[string]string{"status": "503"}, time.Time{})
	if !f.Match(entry) {
		t.Error("expected regex match")
	}
}

func TestFieldFilter_NoMatch(t *testing.T) {
	f, err := NewFieldFilter("status:500")
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "", "", "", map[string]string{"status": "200"}, time.Time{})
	if f.Match(entry) {
		t.Error("expected no match")
	}
}

func TestFieldFilter_InvalidExpression(t *testing.T) {
	_, err := NewFieldFilter("no-colon")
	if err == nil {
		t.Error("expected error for invalid filter expression")
	}
}

func TestFieldFilter_InvalidRegex(t *testing.T) {
	_, err := NewFieldFilter("field:[invalid")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

// --- LevelFilter ---

func TestLevelFilter_ExactMatch(t *testing.T) {
	f := NewLevelFilter("ERROR", true)
	if !f.Match(makeEntry("ERROR", "", "", "", nil, time.Time{})) {
		t.Error("expected exact match for ERROR")
	}
	if f.Match(makeEntry("WARN", "", "", "", nil, time.Time{})) {
		t.Error("expected no match for WARN in exact mode")
	}
	if f.Match(makeEntry("FATAL", "", "", "", nil, time.Time{})) {
		t.Error("expected no match for FATAL in exact mode")
	}
}

func TestLevelFilter_MinLevel(t *testing.T) {
	f := NewLevelFilter("WARN", false)
	if f.Match(makeEntry("INFO", "", "", "", nil, time.Time{})) {
		t.Error("INFO should not match WARN+ filter")
	}
	if !f.Match(makeEntry("WARN", "", "", "", nil, time.Time{})) {
		t.Error("WARN should match WARN+ filter")
	}
	if !f.Match(makeEntry("ERROR", "", "", "", nil, time.Time{})) {
		t.Error("ERROR should match WARN+ filter")
	}
	if !f.Match(makeEntry("FATAL", "", "", "", nil, time.Time{})) {
		t.Error("FATAL should match WARN+ filter")
	}
}

func TestLevelFilter_EmptyLevel(t *testing.T) {
	f := NewLevelFilter("ERROR", true)
	if f.Match(makeEntry("", "", "", "", nil, time.Time{})) {
		t.Error("empty level should not match")
	}
}

// --- RegexFilter ---

func TestRegexFilter_Match(t *testing.T) {
	f, err := NewRegexFilter(`\d{3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "", "", "Connection from 192.168.1.10", nil, time.Time{})
	if !f.Match(entry) {
		t.Error("expected regex match on raw line")
	}
}

func TestRegexFilter_NoMatch(t *testing.T) {
	f, err := NewRegexFilter(`\d{3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	if err != nil {
		t.Fatal(err)
	}
	entry := makeEntry("", "", "", "Application started", nil, time.Time{})
	if f.Match(entry) {
		t.Error("expected no match")
	}
}

func TestRegexFilter_Invalid(t *testing.T) {
	_, err := NewRegexFilter(`[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

// --- TimeRangeFilter ---

func TestTimeRangeFilter_InRange(t *testing.T) {
	from := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	f := NewTimeRangeFilter(from, to)

	ts := time.Date(2024, 1, 15, 8, 30, 0, 0, time.UTC)
	entry := makeEntry("", "", "", "", nil, ts)
	if !f.Match(entry) {
		t.Error("expected match for timestamp within range")
	}
}

func TestTimeRangeFilter_BeforeRange(t *testing.T) {
	from := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	f := NewTimeRangeFilter(from, time.Time{})

	ts := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	entry := makeEntry("", "", "", "", nil, ts)
	if f.Match(entry) {
		t.Error("expected no match for timestamp before range")
	}
}

func TestTimeRangeFilter_AfterRange(t *testing.T) {
	to := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	f := NewTimeRangeFilter(time.Time{}, to)

	ts := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	entry := makeEntry("", "", "", "", nil, ts)
	if f.Match(entry) {
		t.Error("expected no match for timestamp after range")
	}
}

func TestTimeRangeFilter_ZeroTimestamp(t *testing.T) {
	from := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	f := NewTimeRangeFilter(from, time.Time{})

	entry := makeEntry("", "", "", "", nil, time.Time{})
	if f.Match(entry) {
		t.Error("expected no match for zero timestamp")
	}
}

// --- Chain ---

func TestChain_EmptyMatchesAll(t *testing.T) {
	c := NewChain()
	entry := makeEntry("INFO", "hello", "", "", nil, time.Time{})
	if !c.Match(entry) {
		t.Error("empty chain should match everything")
	}
	if !c.IsEmpty() {
		t.Error("expected IsEmpty() to return true")
	}
}

func TestChain_AllMustMatch(t *testing.T) {
	c := NewChain()
	c.Add(NewKeywordFilter("error"))
	c.Add(NewLevelFilter("ERROR", true))

	// Matches both: keyword "error" in message AND level is ERROR.
	entry := makeEntry("ERROR", "An error occurred", "", "An error occurred", nil, time.Time{})
	if !c.Match(entry) {
		t.Error("expected match when all filters match")
	}

	// Matches keyword but not level.
	entry2 := makeEntry("INFO", "An error occurred", "", "An error occurred", nil, time.Time{})
	if c.Match(entry2) {
		t.Error("expected no match when level filter fails")
	}
}

// --- BuildChain ---

func TestBuildChain_Empty(t *testing.T) {
	chain, err := BuildChain("", nil, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !chain.IsEmpty() {
		t.Error("expected empty chain for no filters")
	}
}

func TestBuildChain_AllOptions(t *testing.T) {
	chain, err := BuildChain(
		"timeout",
		[]string{"status:500"},
		"ERROR",
		`\d+ms`,
		"2024-01-15T08:00:00Z",
		"2024-01-15T09:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	if chain.IsEmpty() {
		t.Error("expected non-empty chain")
	}
	// Should have 5 filters: keyword, field, level, regex, time range.
	if len(chain.filters) != 5 {
		t.Errorf("expected 5 filters, got %d", len(chain.filters))
	}
}

func TestBuildChain_InvalidTimeFrom(t *testing.T) {
	_, err := BuildChain("", nil, "", "", "not-a-time", "")
	if err == nil {
		t.Error("expected error for invalid --from time")
	}
}

func TestBuildChain_InvalidTimeTo(t *testing.T) {
	_, err := BuildChain("", nil, "", "", "", "not-a-time")
	if err == nil {
		t.Error("expected error for invalid --to time")
	}
}

func TestBuildChain_InvalidFilter(t *testing.T) {
	_, err := BuildChain("", []string{"no-colon"}, "", "", "", "")
	if err == nil {
		t.Error("expected error for invalid filter expression")
	}
}

func TestBuildChain_InvalidRegex(t *testing.T) {
	_, err := BuildChain("", nil, "", "[invalid", "", "")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}
