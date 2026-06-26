package parser

import (
	"bufio"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// ConcurrentParser wraps a format-specific parser and processes the input
// using multiple goroutines. It splits the input into chunks of lines,
// parses each chunk concurrently, then merges results in line-number order.
//
// For single-worker mode, it delegates directly to the underlying parser
// with zero overhead.
type ConcurrentParser struct {
	format  Format
	workers int
}

// NewConcurrentParser creates a concurrent parser. If workers <= 1 or the
// format doesn't support concurrent parsing, it falls back to sequential.
// A workers value of 0 means auto-detect (runtime.NumCPU()).
func NewConcurrentParser(format Format, workers int) Parser {
	if workers == 0 {
		workers = runtime.NumCPU()
	}
	if workers <= 1 {
		return NewParser(format)
	}
	return &ConcurrentParser{format: format, workers: workers}
}

// chunk holds a batch of raw lines to be parsed by a single worker.
type chunk struct {
	id       int
	lines    []string
	startNum int // 1-based line number of the first line in this chunk
}

// parsedChunk holds the results from parsing one chunk.
type parsedChunk struct {
	id      int
	entries []LogEntry
	errs    []error
}

// Parse reads all input, splits into chunks, parses concurrently, and
// merges results in order. Both returned channels are closed when done.
func (p *ConcurrentParser) Parse(reader io.Reader) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry, 256)
	errs := make(chan error, 64)

	go func() {
		defer close(entries)
		defer close(errs)

		// Read all lines from the input.
		var lines []string
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			errs <- err
			return
		}

		if len(lines) == 0 {
			return
		}

		// Split lines into chunks for parallel processing.
		chunks := p.splitChunks(lines)

		// Parse chunks concurrently.
		results := make([]parsedChunk, len(chunks))
		var wg sync.WaitGroup
		wg.Add(len(chunks))

		for i, c := range chunks {
			go func(idx int, ch chunk) {
				defer wg.Done()
				results[idx] = p.parseChunk(ch)
			}(i, c)
		}

		wg.Wait()

		// Sort results by chunk ID to maintain order.
		sort.Slice(results, func(i, j int) bool {
			return results[i].id < results[j].id
		})

		// Emit all entries and errors in order.
		for _, r := range results {
			for _, e := range r.errs {
				errs <- e
			}
			for _, entry := range r.entries {
				entries <- entry
			}
		}
	}()

	return entries, errs
}

// splitChunks divides the lines into roughly equal chunks for each worker.
func (p *ConcurrentParser) splitChunks(lines []string) []chunk {
	total := len(lines)
	chunkSize := (total + p.workers - 1) / p.workers
	if chunkSize < 10 {
		chunkSize = total // Don't bother splitting tiny inputs.
	}

	var chunks []chunk
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		chunks = append(chunks, chunk{
			id:       len(chunks),
			lines:    lines[i:end],
			startNum: i + 1, // 1-based
		})
	}
	return chunks
}

// parseChunk parses a single chunk using the format-specific parser.
func (p *ConcurrentParser) parseChunk(c chunk) parsedChunk {
	// Reconstruct the chunk as a reader for the underlying parser.
	content := strings.Join(c.lines, "\n")
	if content != "" {
		content += "\n"
	}
	reader := strings.NewReader(content)

	parser := NewParser(p.format)
	entryCh, errCh := parser.Parse(reader)

	result := parsedChunk{id: c.id}

	// Drain errors in a separate goroutine.
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for e := range errCh {
			result.errs = append(result.errs, e)
		}
	}()

	// Collect entries with corrected line numbers.
	for entry := range entryCh {
		// The underlying parser numbers lines starting from 1 within the chunk.
		// Adjust to the global line number.
		entry.LineNum = c.startNum + entry.LineNum - 1
		result.entries = append(result.entries, entry)
	}

	errWg.Wait()
	return result
}
