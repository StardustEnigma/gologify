<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/CLI-Terminal-black?style=for-the-badge&logo=gnometerminal&logoColor=white" alt="CLI">
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-informational?style=for-the-badge" alt="Platform">
</p>

<h1 align="center">⚡ GoLogify</h1>

<p align="center">
  <strong>Stop <code>grep</code>-ing through logs. Start understanding them.</strong>
</p>

<p align="center">
  A blazing-fast, zero-dependency CLI tool that parses, searches, filters, aggregates, and exports log files — all from your terminal.
</p>

<p align="center">
  <a href="https://github.com/StardustEnigma/gologify/actions/workflows/ci.yml"><img src="https://github.com/StardustEnigma/gologify/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/StardustEnigma/gologify/releases"><img src="https://img.shields.io/github/v/release/StardustEnigma/gologify?include_prereleases&style=flat-square" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/StardustEnigma/gologify"><img src="https://pkg.go.dev/badge/github.com/StardustEnigma/gologify.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/StardustEnigma/gologify?style=flat-square" alt="License: MIT"></a>
  <a href="https://goreportcard.com/report/github.com/StardustEnigma/gologify"><img src="https://goreportcard.com/badge/github.com/StardustEnigma/gologify?style=flat-square" alt="Go Report Card"></a>
</p>

---

## The Problem

You're debugging a production outage at 2 AM. You have a 500MB log file. You need answers **now**.

```bash
# This is what your night currently looks like:
grep "ERROR" app.log | grep "2024-01-15" | awk '{print $5}' | sort | uniq -c | sort -rn | head -20
# ↑ Did that even work? Who knows. Good luck remembering this next time.
```

```bash
# This is what it should look like:
gologify analyze app.log --level ERROR --aggregate --top-errors 20
```

**GoLogify** replaces `grep | awk | sed | sort | uniq` chains with a single, intuitive command that actually tells you what's happening in your logs.

---

## ✨ Features

| Feature | Why It Matters |
|---------|---------------|
| 🔍 **Smart Format Detection** | Automatically detects JSON, plain text, CSV, and syslog formats. No `--format` flag needed. |
| 🎯 **Powerful Filtering** | Keyword search, field filters, regex, log levels, time ranges — all composable with AND logic. |
| 📊 **Built-in Analytics** | Level distribution, top errors, top IPs, numeric field statistics (min/max/avg) — no piping to `awk`. |
| 🌈 **Beautiful Output** | Color-coded terminal tables, JSON for scripting, CSV for spreadsheets, raw for piping. |
| ⚡ **Concurrent Processing** | Leverages Go's goroutines to parse large files in parallel across CPU cores. |
| 📦 **Zero Dependencies** | Single binary. Download → Run. No Docker, no `pip install`, no runtime. |
| 🔄 **Real-time Tailing** | Follow logs live with filtering and keyword highlighting. Like `tail -f`, but useful. |
| 📤 **Export Engine** | Filter logs and export matches to JSON, CSV, or raw files in one command. |

---

## 🚀 Quick Start

### Install

**Download binary** (recommended):

```bash
# Linux (amd64)
curl -sL https://github.com/StardustEnigma/gologify/releases/latest/download/gologify_linux_amd64.tar.gz | tar xz
sudo mv gologify /usr/local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/StardustEnigma/gologify/releases/latest/download/gologify_darwin_arm64.tar.gz | tar xz
sudo mv gologify /usr/local/bin/

# Windows — download .zip from Releases page
```

**Or install with Go:**

```bash
go install github.com/StardustEnigma/gologify@latest
```

**Or build from source:**

```bash
git clone https://github.com/StardustEnigma/gologify.git
cd gologify
make build    # → produces ./gologify binary
```

### Your First Analysis

```bash
# See what's in a log file (auto-detects format)
gologify analyze app.log

# Find all errors
gologify analyze app.log --level ERROR

# Get a full statistical breakdown
gologify stats app.log

# Watch logs in real-time
gologify tail app.log --follow --highlight "ERROR"
```

---

## 📖 Usage

### `analyze` — Parse, filter, and display log entries

The primary command. Parses any log file, applies your filters, and shows results.

```bash
# Basic analysis — auto-detects format, shows all entries
gologify analyze app.log

# Search for a keyword across ALL fields
gologify analyze app.log --search "timeout"

# Filter by specific field values (supports regex!)
gologify analyze app.log --filter "status:5[0-9]{2}"

# Combine multiple filters (AND logic)
gologify analyze app.log --filter "level:ERROR" --filter "service:api"

# Filter by log level
gologify analyze app.log --level ERROR

# Filter by time range
gologify analyze app.log --from "2024-01-15T08:00:00Z" --to "2024-01-15T09:00:00Z"

# Show aggregated statistics
gologify analyze app.log --aggregate

# Group results by any field
gologify analyze app.log --aggregate --group-by level

# Output as JSON (pipe to jq, scripts, etc.)
gologify analyze app.log --search "ERROR" --output json

# Output as CSV (open in Excel, Google Sheets)
gologify analyze app.log --output csv > report.csv

# Use regex on raw log lines
gologify analyze app.log --regex "status=(4|5)\d{2}"

# Parallel processing for large files
gologify analyze huge.log --workers 8
```

**Example — find all errors and output as JSON:**

```
$ gologify analyze app.log --level ERROR --output json

{"line":8,"timestamp":"2024-01-15T08:25:12Z","level":"ERROR","message":"Failed to connect to Redis: connection refused (127.0.0.1:6379)"}
{"line":12,"timestamp":"2024-01-15T08:26:30Z","level":"ERROR","message":"Database query timeout after 30s: SELECT * FROM orders WHERE status='pending'"}
{"line":13,"timestamp":"2024-01-15T08:26:30Z","level":"ERROR","message":"Returning 500 Internal Server Error to client 192.168.1.100"}
{"line":19,"timestamp":"2024-01-15T08:30:15Z","level":"ERROR","message":"TLS handshake failed with client 203.0.113.50: certificate expired"}
```

<details>
<summary><strong>All <code>analyze</code> flags</strong></summary>

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `auto` | Log format: `auto`, `json`, `text`, `csv`, `syslog` |
| `--search` | `-s` | | Search keyword across all fields |
| `--filter` | | | Field filter (`field:pattern`), repeatable |
| `--level` | `-l` | | Filter by exact log level |
| `--regex` | | | Filter by regex on raw line |
| `--from` | | | Filter from time (RFC3339) |
| `--to` | | | Filter until time (RFC3339) |
| `--aggregate` | `-a` | `false` | Show aggregated statistics |
| `--group-by` | `-g` | | Group results by field |
| `--top-ips` | | `0` | Show top N IP addresses |
| `--top-errors` | | `0` | Show top N error messages |
| `--output` | `-o` | `table` | Output format: `table`, `json`, `csv`, `raw` |
| `--highlight` | | | Highlight matching text |
| `--limit` | `-n` | `0` | Max entries to display (0 = unlimited) |
| `--workers` | | `0` | Concurrent workers (0 = auto) |

</details>

---

### `stats` — Instant statistical summary

A shortcut for `analyze --aggregate`. One command to understand your entire log file.

```bash
# Full statistics
gologify stats app.log

# Top 20 IPs hitting your server
gologify stats access.log --top-ips 20

# Stats as JSON (for dashboards, scripts)
gologify stats app.log --output json

# Stats as CSV (for spreadsheets)
gologify stats app.log --output csv > stats.csv

# Filter before computing stats
gologify stats app.log --search "api" --level ERROR
```

**Example output:**

```
═══ Log Analysis Summary ═══

  Matched Entries: 15
  Total Entries:   15
  Time Range:      2024-01-15T08:23:01Z → 2024-01-15T08:30:45Z

─── Level Distribution ───

  Level  Count  Bar
  FATAL  1      ███
  ERROR  4      ███████████████
  WARN   2      ███████
  INFO   8      ██████████████████████████████

─── Top Errors ───

  Rank  Error                    Count
  1     Database query timeout   1
  2     Out of memory            1
  3     Redis connection failed  1
  4     Request failed           1
  5     TLS handshake failed     1

─── Numeric Fields ───

  Field        Count  Min    Max       Avg      Sum
  duration_ms  5      1.00   30001.00  6012.20  30061.00
  status       5      200    500       306      1530
```

---

### `tail` — Real-time log following

Like `tail -f` but with filtering and color highlighting.

```bash
# Show last 10 lines (default)
gologify tail app.log

# Show last 50 lines
gologify tail app.log -n 50

# Follow new lines in real-time (Ctrl+C to stop)
gologify tail app.log --follow

# Follow with keyword highlighting
gologify tail app.log -f --highlight "ERROR"

# Follow with filtering — only see errors
gologify tail app.log -f --search "timeout" --level ERROR
```

---

### `export` — Save filtered results to a file

Filter your logs and write matches directly to a new file.

```bash
# Export all errors as JSON
gologify export app.log --output errors.json --format json --level ERROR

# Export filtered entries as CSV
gologify export access.log --output filtered.csv --format csv --filter "status:500"

# Export with time range
gologify export app.log --output recent.log --format raw --from "2024-01-15T08:00:00Z"

# Export a sample (first 100 matches)
gologify export app.log --output sample.json --format json --limit 100
```

---

### `version` — Version info

```
$ gologify version
GoLogify v0.1.0
  Go:       go1.25.5
  OS/Arch:  linux/amd64
  Compiler: gc
```

---

## 🔎 Supported Log Formats

GoLogify auto-detects the format of your log file. You can also specify it explicitly with `--format`.

| Format | Auto-detected? | Example |
|--------|:-:|---------|
| **JSON Lines** | ✅ | `{"timestamp":"2024-01-15T08:23:01Z","level":"info","msg":"Request processed"}` |
| **Plain Text** | ✅ | `2024-01-15 10:30:45 ERROR Something happened` |
| **CSV** | ✅ | `timestamp,level,message` (header row + data rows) |
| **Syslog (RFC 3164)** | ✅ | `Jan 15 10:30:45 server app[1234]: message` |

### Recognized field names

GoLogify maps common field names to a unified schema, so your logs work regardless of naming conventions:

| Field | Recognized names |
|-------|-----------------|
| **Timestamp** | `timestamp`, `time`, `ts`, `t`, `@timestamp`, `date`, `datetime` |
| **Level** | `level`, `severity`, `lvl`, `loglevel` |
| **Message** | `message`, `msg`, `text`, `description` |
| **Source** | `source`, `service`, `hostname`, `host`, `app` |

Log levels are normalized automatically: `info` → `INFO`, `err` → `ERROR`, `WRN` → `WARN`, `FTL` → `FATAL`, etc.

---

## 🔥 Real-World Scenarios

### 🚨 Production Outage Investigation

```bash
# Step 1: What happened in the last hour?
gologify stats app.log --output json

# Step 2: Find all errors
gologify analyze app.log --level ERROR --from "2024-01-15T08:00:00Z"

# Step 3: Which service is failing?
gologify analyze app.log --level ERROR --aggregate --group-by service

# Step 4: What are the most common errors?
gologify stats app.log --top-errors 20

# Step 5: Export for the incident report
gologify export app.log --output incident.csv --format csv --level ERROR
```

### 🔒 Security Incident Response

```bash
# Find suspicious IPs
gologify stats access.log --top-ips 50 --output csv > suspicious_ips.csv

# Search for attack patterns
gologify analyze access.log --regex "(\.\.\/|SELECT|UNION|DROP)" --output json

# Export all 4xx/5xx responses
gologify export access.log --output errors.csv --format csv --filter "status:^[45]"
```

### 🐌 Performance Debugging

```bash
# Find slow requests (4+ digit durations = 1000ms+)
gologify analyze app.log --filter "duration_ms:^[0-9]{4,}" --output table

# Get duration statistics
gologify analyze app.log --aggregate --output json | jq '.numeric_stats.duration_ms'

# Watch for slow requests in real-time
gologify tail app.log --follow --highlight "slow"
```

### 📋 Daily Operations

```bash
# Morning health check
gologify stats /var/log/syslog --top-errors 10

# Generate daily report
gologify stats app.log --output csv > "report-$(date +%Y-%m-%d).csv"

# Monitor deployments
gologify tail app.log -f --search "deployed" --highlight "ERROR"
```

---

## 🏗️ Architecture

```
gologify/
├── main.go                  # Entry point
├── cmd/                     # CLI commands (Cobra)
│   ├── root.go              # Root command, global flags (--verbose, --no-color)
│   ├── analyze.go           # analyze — parse, filter, aggregate, display
│   ├── stats.go             # stats — quick summary (shortcut for analyze --aggregate)
│   ├── tail.go              # tail — last N lines + follow mode
│   ├── export.go            # export — filter and write to file
│   └── version.go           # version — build info
├── pkg/
│   ├── parser/              # Log format parsers (streaming, channel-based)
│   │   ├── parser.go        # Parser interface, format detection, shared utilities
│   │   ├── json.go          # JSON Lines parser (zerolog, zap, logrus compatible)
│   │   ├── text.go          # Plain text parser (5 regex patterns)
│   │   ├── csv.go           # CSV parser (header-aware, auto-maps columns)
│   │   ├── syslog.go        # RFC 3164 syslog parser
│   │   └── concurrent.go    # Parallel parser (splits input across goroutines)
│   ├── filter/              # Composable filter engine
│   │   └── filter.go        # Keyword, field, level, regex, time range filters
│   ├── aggregator/          # Statistics aggregation
│   │   └── aggregator.go    # Level counts, group-by, top-N, numeric stats
│   └── output/              # Output formatters
│       ├── table.go         # Pretty terminal tables with color-coded levels
│       ├── json.go          # JSON / JSON Lines output
│       ├── csv.go           # CSV output with dynamic headers
│       └── raw.go           # Raw lines with keyword highlighting
├── examples/                # Sample log files for testing
│   ├── sample.json          # JSON Lines log (15 entries)
│   ├── sample.log           # Plain text log (25 entries)
│   ├── sample.csv           # CSV log (16 entries)
│   └── sample.syslog        # Syslog format (15 entries)
├── Makefile                 # Build automation
├── .goreleaser.yml          # Cross-platform release config
└── .github/workflows/
    ├── ci.yml               # CI: test on Linux/macOS/Windows + lint
    └── release.yml          # CD: GoReleaser on tag push
```

### Design Principles

- **Streaming architecture** — Parsers emit entries through Go channels. Files of any size are processed with constant memory.
- **Composable filters** — Filters implement a `Match(entry)` interface and chain with AND logic. Easy to extend.
- **Format-agnostic core** — All parsers produce the same `LogEntry` struct. The rest of the pipeline doesn't care about the original format.
- **Concurrent by default** — Large files are automatically split across CPU cores for parallel parsing.

---

## ⚙️ Global Flags

| Flag | Description |
|------|-------------|
| `--verbose`, `-v` | Enable verbose output (shows parse warnings, auto-detection info) |
| `--no-color` | Disable colored output (for piping, CI environments) |
| `--help`, `-h` | Show help for any command |
| `--version` | Show version |

---

## 🛠️ Development

```bash
# Prerequisites: Go 1.25+

# Clone
git clone https://github.com/StardustEnigma/gologify.git
cd gologify

# Run tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage report
make test-cover

# Run benchmarks
make bench

# Format code
make fmt

# Lint (requires staticcheck)
make lint

# Build for current platform
make build

# Cross-compile for all platforms
make build-all
```

### Running Locally

```bash
# Build and run
make build
./gologify analyze examples/sample.json --aggregate

# Or use go run
go run . analyze examples/sample.log --search "ERROR"
```

---

## 🤝 Contributing

Contributions are welcome! Whether it's a bug fix, new feature, or documentation improvement.

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Write** your code and tests
4. **Test**: `make test` (all tests must pass)
5. **Lint**: `make lint`
6. **Commit**: `git commit -m 'feat: add amazing feature'`
7. **Push**: `git push origin feature/amazing-feature`
8. **Open** a Pull Request

Please follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.

### Ideas for Contributions

- [ ] Additional log format parsers (e.g., Apache access logs, nginx, Docker)
- [ ] `--since` / `--until` relative time filters (`--since 1h`)
- [ ] Interactive TUI mode with live filtering
- [ ] Log file glob support (`gologify analyze *.log`)
- [ ] Plugin system for custom parsers
- [ ] Compressed file support (.gz, .zst)

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <sub>Built with ❤️ in Go. If GoLogify saved you time, consider giving it a ⭐</sub>
</p>
