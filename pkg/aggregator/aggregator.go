// Package aggregator provides streaming statistics aggregation for log entries.
package aggregator

import (
	"sort"
	"strconv"
	"strings"

	"github.com/StardustEnigma/gologify/pkg/parser"
)

// Result holds the aggregated statistics after processing all entries.
type Result struct {
	TotalEntries   int                          // total entries processed
	MatchedEntries int                          // entries that passed filters
	LevelCounts    map[string]int               // count per log level
	GroupCounts    map[string]map[string]int     // field -> value -> count
	NumericStats   map[string]*NumericStat       // field -> statistics
	TopN           []TopEntry                    // for top-N queries
	FirstTimestamp string                        // earliest timestamp seen
	LastTimestamp  string                        // latest timestamp seen
	ErrorMessages  map[string]int               // error message -> count
}

// NumericStat holds min/max/avg/sum for a numeric field.
type NumericStat struct {
	Count int
	Min   float64
	Max   float64
	Sum   float64
	Avg   float64
}

// TopEntry holds a value and its count for top-N displays.
type TopEntry struct {
	Value string
	Count int
}

// Aggregator accumulates statistics as log entries stream through.
type Aggregator struct {
	result      Result
	groupByField string
	topIPsN     int
	topErrorsN  int
	ipCounts    map[string]int
}

// New creates a new Aggregator with the given options.
func New(groupBy string, topIPs, topErrors int) *Aggregator {
	return &Aggregator{
		result: Result{
			LevelCounts:   make(map[string]int),
			GroupCounts:   make(map[string]map[string]int),
			NumericStats:  make(map[string]*NumericStat),
			ErrorMessages: make(map[string]int),
		},
		groupByField: strings.ToLower(groupBy),
		topIPsN:      topIPs,
		topErrorsN:   topErrors,
		ipCounts:     make(map[string]int),
	}
}

// Add processes a single log entry and updates the aggregation state.
func (a *Aggregator) Add(entry parser.LogEntry) {
	a.result.MatchedEntries++

	// Level counts.
	if entry.Level != "" {
		a.result.LevelCounts[entry.Level]++
	}

	// Timestamp tracking.
	if !entry.Timestamp.IsZero() {
		ts := entry.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		if a.result.FirstTimestamp == "" || ts < a.result.FirstTimestamp {
			a.result.FirstTimestamp = ts
		}
		if a.result.LastTimestamp == "" || ts > a.result.LastTimestamp {
			a.result.LastTimestamp = ts
		}
	}

	// Group-by aggregation.
	if a.groupByField != "" {
		value := a.getFieldValue(entry, a.groupByField)
		if value != "" {
			if a.result.GroupCounts[a.groupByField] == nil {
				a.result.GroupCounts[a.groupByField] = make(map[string]int)
			}
			a.result.GroupCounts[a.groupByField][value]++
		}
	}

	// IP tracking.
	if a.topIPsN > 0 {
		ip := a.getFieldValue(entry, "ip")
		if ip == "" {
			ip = a.getFieldValue(entry, "client_ip")
		}
		if ip == "" {
			ip = a.getFieldValue(entry, "remote_addr")
		}
		if ip != "" {
			a.ipCounts[ip]++
		}
	}

	// Error message tracking.
	if entry.Level == "ERROR" || entry.Level == "FATAL" || entry.Level == "PANIC" {
		msg := entry.Message
		if msg == "" {
			msg = entry.Raw
		}
		// Truncate long messages for grouping.
		if len(msg) > 100 {
			msg = msg[:100] + "..."
		}
		a.result.ErrorMessages[msg]++
	}

	// Numeric field detection and statistics.
	for k, v := range entry.Fields {
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			stat, ok := a.result.NumericStats[k]
			if !ok {
				stat = &NumericStat{Min: num, Max: num}
				a.result.NumericStats[k] = stat
			}
			stat.Count++
			stat.Sum += num
			if num < stat.Min {
				stat.Min = num
			}
			if num > stat.Max {
				stat.Max = num
			}
			stat.Avg = stat.Sum / float64(stat.Count)
		}
	}
}

// SetTotal sets the total number of entries processed (before filtering).
func (a *Aggregator) SetTotal(total int) {
	a.result.TotalEntries = total
}

// Result returns the final aggregation result.
func (a *Aggregator) Result() Result {
	// Compute top IPs.
	if a.topIPsN > 0 {
		a.result.TopN = topN(a.ipCounts, a.topIPsN)
	}

	return a.result
}

// TopErrors returns the top N error messages by frequency.
func (a *Aggregator) TopErrors() []TopEntry {
	return topN(a.result.ErrorMessages, a.topErrorsN)
}

// getFieldValue extracts a named field from a LogEntry,
// checking both well-known fields and the Fields map.
func (a *Aggregator) getFieldValue(entry parser.LogEntry, field string) string {
	switch field {
	case "level", "severity":
		return entry.Level
	case "message", "msg":
		return entry.Message
	case "source", "service", "host":
		return entry.Source
	}

	// Case-insensitive lookup in Fields.
	lower := strings.ToLower(field)
	for k, v := range entry.Fields {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

// topN returns the top N entries from a frequency map, sorted descending.
func topN(counts map[string]int, n int) []TopEntry {
	entries := make([]TopEntry, 0, len(counts))
	for k, v := range counts {
		entries = append(entries, TopEntry{Value: k, Count: v})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Value < entries[j].Value
		}
		return entries[i].Count > entries[j].Count
	})

	if n > 0 && n < len(entries) {
		entries = entries[:n]
	}
	return entries
}
