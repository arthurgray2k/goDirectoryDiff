# goDirectoryDiff

`goDirectoryDiff` is a high-performance command-line tool written in Go that recursively compares two directories and outputs changes in standard Git unified diff format.

The project executable CLI binary is **`goDirDiff`**.

## Features
- **Pure Go Standard Library**: Zero external dependencies.
- **Git Unified Diff Format**: Emits standard Git patch format (`diff --git a/... b/...`, `--- a/...`, `+++ b/...`, `@@ -start,len +start,len @@`).
- **File Lifecycle Detection**: Detects modified files, added files (`new file mode 100644`), and deleted files (`deleted file mode 100644`).
- **Binary File Detection**: Automatically detects binary files and prints difference notices without polluting output.
- **Configurable Context**: Custom context lines via `-c` / `--context`.
- **Path Filtering**: Focus on specific files or subfolders via `-f` / `--filter` (gracefully ignored if invalid).
- **Diff Export**: Export diff files via `-e` / `--export` or custom paths via `-o` / `--export-path`. Defaults to `exported_diff/` in the project directory.
- **SHA-256 Integrity Verification**: Generates companion `<file>.sha256` checksum files in standard GNU coreutils format alongside exported diffs for cryptographic integrity checking. Automatically validated when applying patches.
- **Bidirectional Diff Patching**: Apply patches in forward (`lr`, `l->r`) or reverse (`rl`, `r->l`) direction (`-a` / `--apply`, `-d` / `--direction`, `-p` / `--patch`, `-t` / `--target`).
- **Direct Directory Synchronization**: Seamlessly sync changes directly between two directory trees in either direction.
- **Colorized Output**: Optional ANSI terminal color highlighting (`--color`).
- **Included Sample Demonstration**: Pre-packaged example directories (`examples/dir_v1` and `examples/dir_v2`) to demonstrate directory diffing immediately.

## Architecture
```
goDirectoryDiff/
├── cmd/
│   └── godirdiff/
│       └── main.go           # CLI entry point, executable name: goDirDiff
├── internal/
│   └── diff/
│       ├── diff.go           # Recursive directory traversal & diff generator
│       └── diff_test.go      # Comprehensive test suite (>85% coverage)
├── examples/
│   ├── dir_v1/               # Sample directory version 1
│   └── dir_v2/               # Sample directory version 2
├── go.mod                    # Module definition (Go 1.26)
├── README.md
└── USAGE.md
```

## Building
```bash
go build -o goDirDiff ./cmd/godirdiff
```

## Testing
```bash
go test -v -cover ./...
```
