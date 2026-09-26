# goDirectoryDiff

`goDirectoryDiff` is a high-performance command-line tool written in Go that recursively compares two directories and outputs changes in standard Git unified diff format.

## Features
- **Pure Go Standard Library**: Zero external dependencies.
- **Git Unified Diff Format**: Emits `diff --git a/... b/...`, `--- a/...`, `+++ b/...`, and hunk markers (`@@ -start,len +start,len @@`).
- **File Lifecycle Awareness**: Detects modified files, added files (`new file mode 100644`), and deleted files (`deleted file mode 100644`).
- **Binary File Detection**: Automatically detects binary files and prints difference notices without polluting output.
- **Configurable Context**: Custom context lines via `-c` / `--context`.
- **Colorized Output**: Optional ANSI terminal color highlighting (`--color`).

## Architecture
```
goDirectoryDiff/
├── cmd/
│   └── godirectorydiff/
│       └── main.go           # CLI entry point, argument parsing, exit codes
├── internal/
│   └── diff/
│       ├── diff.go           # Recursive directory traversal & diff generator
│       └── diff_test.go      # Comprehensive test suite (>85% coverage)
├── go.mod                    # Module definition (Go 1.26)
├── README.md
└── USAGE.md
```

## Building
```bash
go build ./cmd/godirectorydiff
```

## Testing
```bash
go test -v -cover ./...
```
