package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/StardustEnigma/gologify/pkg/filter"
	"github.com/StardustEnigma/gologify/pkg/output"
	"github.com/StardustEnigma/gologify/pkg/parser"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	tailFollow      bool
	tailLines       int
	tailHighlight   string
	tailFormat      string
	tailSearch      string
	tailLevel       string
	tailColorize    bool
)

var tailCmd = &cobra.Command{
	Use:   "tail [file]",
	Short: "Tail a log file with optional follow mode",
	Long: `Display the last N lines of a log file, optionally following
new lines as they are appended (like tail -f).

Examples:
  gologify tail app.log
  gologify tail app.log -n 50
  gologify tail app.log --follow --highlight "ERROR"
  gologify tail app.log -f --search "timeout" --level ERROR`,
	Args: cobra.ExactArgs(1),
	RunE: runTail,
}

func init() {
	rootCmd.AddCommand(tailCmd)

	tailCmd.Flags().BoolVarP(&tailFollow, "follow", "f", false, "follow new lines (like tail -f)")
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 10, "number of lines to show")
	tailCmd.Flags().StringVar(&tailHighlight, "highlight", "", "highlight matching text")
	tailCmd.Flags().StringVar(&tailFormat, "format", "auto", "log format: auto, json, text, csv, syslog")
	tailCmd.Flags().StringVarP(&tailSearch, "search", "s", "", "filter by keyword")
	tailCmd.Flags().StringVarP(&tailLevel, "level", "l", "", "filter by log level")
	tailCmd.Flags().BoolVar(&tailColorize, "colorize", true, "colorize log levels in output")
}

func runTail(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", filePath)
	}

	// Build filter if needed.
	chain, err := filter.BuildChain(tailSearch, nil, tailLevel, "", "", "")
	if err != nil {
		return err
	}

	// Read last N lines.
	lastLines, err := readLastLines(file, tailLines)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Display the last lines.
	for _, line := range lastLines {
		if !chain.IsEmpty() {
			entry := parser.LogEntry{Raw: line, Message: line}
			if !chain.Match(entry) {
				continue
			}
		}
		printTailLine(line)
	}

	// Follow mode.
	if tailFollow {
		return followFile(filePath, chain)
	}

	return nil
}

// readLastLines reads the last N lines from a file using backward seeking.
func readLastLines(file *os.File, n int) ([]string, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		return nil, nil
	}

	// Read from end of file backward to find last N newlines.
	const chunkSize = 8192
	fileSize := info.Size()
	var lines []string
	remaining := make([]byte, 0)

	offset := fileSize
	for offset > 0 && len(lines) <= n {
		readSize := int64(chunkSize)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize

		chunk := make([]byte, readSize)
		_, err := file.ReadAt(chunk, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Prepend to remaining.
		chunk = append(chunk, remaining...)
		remaining = nil

		// Split into lines.
		parts := strings.Split(string(chunk), "\n")

		// First part may be partial (we split mid-line), save it.
		if offset > 0 {
			remaining = []byte(parts[0])
			parts = parts[1:]
		}

		// Prepend found lines.
		for i := len(parts) - 1; i >= 0; i-- {
			if strings.TrimSpace(parts[i]) != "" {
				lines = append([]string{parts[i]}, lines...)
			}
		}
	}

	// Handle remaining partial first line.
	if len(remaining) > 0 && strings.TrimSpace(string(remaining)) != "" {
		lines = append([]string{string(remaining)}, lines...)
	}

	// Return only last N.
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines, nil
}

// followFile watches a file for new content and prints new lines.
func followFile(filePath string, chain *filter.Chain) error {
	// Set up graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n"+color.YellowString("→ stopped following"))
		cancel()
	}()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file for following: %w", err)
	}
	defer file.Close()

	// Seek to end.
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seeking to end: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s following %s (Ctrl+C to stop)\n",
		color.CyanString("→"), filePath)

	reader := bufio.NewReader(file)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				line = strings.TrimRight(line, "\n\r")
				if strings.TrimSpace(line) == "" {
					continue
				}

				// Apply filter.
				if !chain.IsEmpty() {
					entry := parser.LogEntry{
						Raw:     line,
						Message: line,
					}
					if !chain.Match(entry) {
						continue
					}
				}

				printTailLine(line)
			}
		}
	}
}

// printTailLine prints a single line with optional highlighting and level coloring.
func printTailLine(line string) {
	displayed := line

	// Colorize log levels.
	if tailColorize {
		displayed = output.FormatLevelColor(displayed)
	}

	// Apply highlighting.
	if tailHighlight != "" {
		displayed = output.HighlightText(displayed, tailHighlight)
	}

	fmt.Println(displayed)
}
