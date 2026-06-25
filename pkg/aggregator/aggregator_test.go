package aggregator

import (
	"testing"
	"time"

	"github.com/StardustEnigma/gologify/pkg/parser"
)

func makeEntry(level, message, source string, fields map[string]string, ts time.Time) parser.LogEntry {
	if fields == nil {
		fields = make(map[string]string)
	}
	return parser.LogEntry{
		Timestamp: ts,
		Level:     level,
		Message:   message,
		Source:    source,
		Fields:    fields,
	}
}

func TestAggregator_LevelCounts(t *testing.T) {
	agg := New("", 0, 10)

	agg.Add(makeEntry("INFO", "msg1", "", nil, time.Time{}))
	agg.Add(makeEntry("INFO", "msg2", "", nil, time.Time{}))
	agg.Add(makeEntry("ERROR", "msg3", "", nil, time.Time{}))
	agg.Add(makeEntry("WARN", "msg4", "", nil, time.Time{}))

	result := agg.Result()

	if result.MatchedEntries != 4 {
		t.Errorf("MatchedEntries = %d, want 4", result.MatchedEntries)
	}
	if result.LevelCounts["INFO"] != 2 {
		t.Errorf("LevelCounts[INFO] = %d, want 2", result.LevelCounts["INFO"])
	}
	if result.LevelCounts["ERROR"] != 1 {
		t.Errorf("LevelCounts[ERROR] = %d, want 1", result.LevelCounts["ERROR"])
	}
	if result.LevelCounts["WARN"] != 1 {
		t.Errorf("LevelCounts[WARN] = %d, want 1", result.LevelCounts["WARN"])
	}
}

func TestAggregator_TimestampTracking(t *testing.T) {
	agg := New("", 0, 10)

	ts1 := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	ts3 := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	agg.Add(makeEntry("INFO", "msg1", "", nil, ts1))
	agg.Add(makeEntry("INFO", "msg2", "", nil, ts2))
	agg.Add(makeEntry("INFO", "msg3", "", nil, ts3))

	result := agg.Result()

	if result.FirstTimestamp != "2024-01-15T08:00:00Z" {
		t.Errorf("FirstTimestamp = %q, want 2024-01-15T08:00:00Z", result.FirstTimestamp)
	}
	if result.LastTimestamp != "2024-01-15T10:00:00Z" {
		t.Errorf("LastTimestamp = %q, want 2024-01-15T10:00:00Z", result.LastTimestamp)
	}
}

func TestAggregator_GroupBy(t *testing.T) {
	agg := New("level", 0, 10)

	agg.Add(makeEntry("INFO", "msg1", "", nil, time.Time{}))
	agg.Add(makeEntry("INFO", "msg2", "", nil, time.Time{}))
	agg.Add(makeEntry("ERROR", "msg3", "", nil, time.Time{}))

	result := agg.Result()

	if result.GroupCounts["level"]["INFO"] != 2 {
		t.Errorf("GroupCounts[level][INFO] = %d, want 2", result.GroupCounts["level"]["INFO"])
	}
	if result.GroupCounts["level"]["ERROR"] != 1 {
		t.Errorf("GroupCounts[level][ERROR] = %d, want 1", result.GroupCounts["level"]["ERROR"])
	}
}

func TestAggregator_TopIPs(t *testing.T) {
	agg := New("", 3, 10)

	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "192.168.1.10"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "192.168.1.10"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "192.168.1.10"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "10.0.0.1"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "10.0.0.1"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"ip": "172.16.0.1"}, time.Time{}))

	result := agg.Result()

	if len(result.TopN) != 3 {
		t.Fatalf("TopN length = %d, want 3", len(result.TopN))
	}
	if result.TopN[0].Value != "192.168.1.10" || result.TopN[0].Count != 3 {
		t.Errorf("TopN[0] = %v, want 192.168.1.10:3", result.TopN[0])
	}
	if result.TopN[1].Value != "10.0.0.1" || result.TopN[1].Count != 2 {
		t.Errorf("TopN[1] = %v, want 10.0.0.1:2", result.TopN[1])
	}
}

func TestAggregator_ErrorTracking(t *testing.T) {
	agg := New("", 0, 5)

	agg.Add(makeEntry("ERROR", "Connection failed", "", nil, time.Time{}))
	agg.Add(makeEntry("ERROR", "Connection failed", "", nil, time.Time{}))
	agg.Add(makeEntry("ERROR", "Timeout", "", nil, time.Time{}))
	agg.Add(makeEntry("FATAL", "Out of memory", "", nil, time.Time{}))
	agg.Add(makeEntry("INFO", "Request OK", "", nil, time.Time{})) // not an error

	topErrs := agg.TopErrors()

	if len(topErrs) != 3 {
		t.Fatalf("TopErrors length = %d, want 3", len(topErrs))
	}
	if topErrs[0].Value != "Connection failed" || topErrs[0].Count != 2 {
		t.Errorf("TopErrors[0] = %v, want Connection failed:2", topErrs[0])
	}
}

func TestAggregator_NumericStats(t *testing.T) {
	agg := New("", 0, 10)

	agg.Add(makeEntry("INFO", "", "", map[string]string{"duration_ms": "10"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"duration_ms": "20"}, time.Time{}))
	agg.Add(makeEntry("INFO", "", "", map[string]string{"duration_ms": "30"}, time.Time{}))

	result := agg.Result()

	stat, ok := result.NumericStats["duration_ms"]
	if !ok {
		t.Fatal("expected numeric stats for duration_ms")
	}
	if stat.Count != 3 {
		t.Errorf("Count = %d, want 3", stat.Count)
	}
	if stat.Min != 10 {
		t.Errorf("Min = %f, want 10", stat.Min)
	}
	if stat.Max != 30 {
		t.Errorf("Max = %f, want 30", stat.Max)
	}
	if stat.Sum != 60 {
		t.Errorf("Sum = %f, want 60", stat.Sum)
	}
	if stat.Avg != 20 {
		t.Errorf("Avg = %f, want 20", stat.Avg)
	}
}

func TestAggregator_SetTotal(t *testing.T) {
	agg := New("", 0, 10)
	agg.Add(makeEntry("INFO", "", "", nil, time.Time{}))
	agg.SetTotal(100)

	result := agg.Result()
	if result.TotalEntries != 100 {
		t.Errorf("TotalEntries = %d, want 100", result.TotalEntries)
	}
	if result.MatchedEntries != 1 {
		t.Errorf("MatchedEntries = %d, want 1", result.MatchedEntries)
	}
}

func TestAggregator_LongErrorTruncation(t *testing.T) {
	agg := New("", 0, 10)

	longMsg := ""
	for i := 0; i < 150; i++ {
		longMsg += "x"
	}
	agg.Add(makeEntry("ERROR", longMsg, "", nil, time.Time{}))

	topErrs := agg.TopErrors()
	if len(topErrs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(topErrs))
	}
	if len(topErrs[0].Value) != 103 { // 100 chars + "..."
		t.Errorf("truncated error length = %d, want 103", len(topErrs[0].Value))
	}
}

func TestTopN_Sorting(t *testing.T) {
	counts := map[string]int{
		"a": 5,
		"b": 10,
		"c": 3,
		"d": 10, // tie with b, should sort alphabetically
	}

	result := topN(counts, 3)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result[0].Count != 10 {
		t.Errorf("result[0].Count = %d, want 10", result[0].Count)
	}
	if result[1].Count != 10 {
		t.Errorf("result[1].Count = %d, want 10", result[1].Count)
	}
	// Tie-break should be alphabetical.
	if result[0].Value != "b" || result[1].Value != "d" {
		t.Errorf("tie-break order wrong: %q, %q", result[0].Value, result[1].Value)
	}
	if result[2].Count != 5 {
		t.Errorf("result[2].Count = %d, want 5", result[2].Count)
	}
}

func TestTopN_LimitExceedsSize(t *testing.T) {
	counts := map[string]int{"a": 1, "b": 2}
	result := topN(counts, 10)
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestAggregator_IPFallbackFields(t *testing.T) {
	agg := New("", 5, 10)

	// Test "client_ip" fallback.
	agg.Add(makeEntry("INFO", "", "", map[string]string{"client_ip": "10.0.0.1"}, time.Time{}))
	// Test "remote_addr" fallback.
	agg.Add(makeEntry("INFO", "", "", map[string]string{"remote_addr": "10.0.0.2"}, time.Time{}))

	result := agg.Result()
	if len(result.TopN) != 2 {
		t.Errorf("expected 2 IPs from fallback fields, got %d", len(result.TopN))
	}
}
