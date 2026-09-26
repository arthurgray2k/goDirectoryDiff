# USAGE: goDirDiff

`goDirDiff` is a versatile directory comparison, diff export, and bidirectional patch application tool.

---

## Command Syntax Overview

```bash
# 1. Compare two directories
goDirDiff [options] <dir1> <dir2>

# 2. Apply an exported diff patch file to a target directory
goDirDiff -a [-d rl|lr] [-p <patch_file>] -t <target_dir>
```

---

## Options & Flags Reference

| Flag | Shorthand | Description | Default |
|---|---|---|---|
| `-apply` | `-a` | Enable patch application mode | `false` |
| `-patch <file>` | `-p <file>` | Path to patch file to apply | `exported_diff/diff.patch` |
| `-target <dir>` | `-t <dir>` | Target directory to apply patch to (required for apply) | `""` |
| `-direction <mode>` | `-d <mode>` | Patch direction: `rl` (right/dir2 changes) or `lr` (left/dir1 changes) | `rl` |
| `-export` | `-e` | Export diff to default directory (`exported_diff/`) | `false` |
| `-export-path <path>`| `-o <path>` | Custom destination file or directory path for export | `""` |
| `-filter <path>` | `-f <path>` | Scope diff to a specific file or subfolder | `""` |
| `-context <n>` | `-c <n>` | Number of context lines surrounding diff hunks | `3` |
| `-color` | N/A | Enable ANSI colored terminal diff output | `false` |
| `-help` | `-h` | Display command help and usage instructions | |

---

## Exit Codes

| Code | Status | Meaning |
|---|---|---|
| `0` | Success / Clean | In comparison mode: directories are identical. In patch mode: patch applied successfully. |
| `1` | Differences Found | In comparison mode: differences were found and output to stdout. |
| `2` | Error | Execution failed (e.g. invalid arguments, missing directories, checksum mismatch). |

---

## CLI Decision Flowchart

```mermaid
flowchart TD
    Start(["goDirDiff Invocation"]) --> CheckApply{"Is -a / -p / -t specified?"}

    CheckApply -- "No (Comparison Mode)" --> CheckDirs{"Are 2 directories provided?"}
    CheckDirs -- "No" --> UsageErr["Show Usage & Exit 2"]
    CheckDirs -- "Yes" --> RunDiff["Recursively compare dir1 and dir2"]
    RunDiff --> FilterOpt{"Is -f / --filter given?"}
    FilterOpt -- "Yes" --> FilterScope["Filter differences to path"]
    FilterOpt -- "No" --> ExportOpt
    FilterScope --> ExportOpt{"Is -e or -o given?"}
    ExportOpt -- "Yes" --> SavePatch["Save diff.patch & generate companion .sha256"]
    ExportOpt -- "No" --> StdoutDiff["Print Git unified diff to stdout"]
    SavePatch --> StdoutDiff
    StdoutDiff --> DiffStatus{"Any differences?"}
    DiffStatus -- "Yes" --> Exit1["Exit 1 (Diff Found)"]
    DiffStatus -- "No" --> Exit0["Exit 0 (Identical)"]

    CheckApply -- "Yes (Patch Mode)" --> ValidateTarget{"Is Target Directory provided?"}
    ValidateTarget -- "No" --> MissingTargetErr["Print Error & Exit 2"]
    ValidateTarget -- "Yes" --> ValidatePatch["Check companion .sha256 integrity"]
    ValidatePatch -- "Hash Mismatch" --> ChecksumErr["Print Checksum Error & Exit 2"]
    ValidatePatch -- "Valid / No Hash" --> ParseHunks["ParsePatch (Hunks & File Patches)"]
    ParseHunks --> CheckDir{"Direction -d"}
    CheckDir -- "rl (default)" --> ApplyRL["Bring in right (dir2) changes to target directory"]
    CheckDir -- "lr" --> ApplyLR["Bring in left (dir1) changes to target directory"]
    ApplyRL --> PatchDone["Print summary of modified/added/deleted files"]
    ApplyLR --> PatchDone
    PatchDone --> Exit0
```

---

## Feature Guides & Practical Examples

### 1. Basic Directory Comparison
Compare two folders and display standard Git unified diffs on stdout:

```bash
./goDirDiff examples/dir_v1 examples/dir_v2
```
*Output snippet:*
```diff
diff --git a/README.txt b/README.txt
--- a/README.txt
+++ b/README.txt
@@ -1,7 +1,8 @@
 Project Alpha
-Version 1.0.0
+Version 2.0.0
 Author: Team Alpha
...
```

---

### 2. Colorized Terminal Output
Highlight additions in green and deletions in red for enhanced terminal readability:

```bash
./goDirDiff --color examples/dir_v1 examples/dir_v2
```

---

### 3. Custom Context Lines
Adjust the amount of surrounding context lines displayed around changed hunks:

```bash
# 5 lines of context
./goDirDiff -c 5 examples/dir_v1 examples/dir_v2

# 1 line of context (compact view)
./goDirDiff -c 1 examples/dir_v1 examples/dir_v2
```

---

### 4. Scoped Filtering (`-f` / `--filter`)
Focus only on changes within a specific file or subfolder:

```bash
# Compare only config.json
./goDirDiff -f config.json examples/dir_v1 examples/dir_v2

# Compare only a subfolder
./goDirDiff -f services/api examples/dir_v1 examples/dir_v2
```
*Note: If the filtered path does not exist in the comparison, `goDirDiff` gracefully falls back to showing all changes.*

---

### 5. Exporting Diffs & Generating SHA-256 Checksums

#### Export to Default Location (`exported_diff/`)
```bash
./goDirDiff -e examples/dir_v1 examples/dir_v2
```
Generates:
- `exported_diff/diff.patch`: Git-compatible unified diff file.
- `exported_diff/diff.patch.sha256`: Companion cryptographic checksum file in standard GNU format.

#### Export to a Custom File or Directory
```bash
# Export to a custom patch file
./goDirDiff -o builds/release_v2.patch examples/dir_v1 examples/dir_v2

# Export to a specific directory (writes diff.patch and diff.patch.sha256 inside)
./goDirDiff -o /tmp/patches/ examples/dir_v1 examples/dir_v2
```

#### Verifying Checksums
The companion checksum file can be verified using standard system utilities:
```bash
cd exported_diff
sha256sum -c diff.patch.sha256
# Output: diff.patch: OK
```

---

### 6. Bidirectional Patch Application

Apply mode is strictly designed to apply an exported diff (`exported_diff/diff.patch`) and verify its companion `.sha256` checksum to a target directory provided by the user.

#### Right / Dir2 Application (`-d rl`, Default)
Brings in **right** (or bottom) changes that belong to `dir2` into the target directory:

```bash
# Apply diff.patch to /path/to/target (defaults to direction: rl)
./goDirDiff -a -p exported_diff/diff.patch -t /path/to/target

# Explicit right syntax
./goDirDiff -a -d rl -p exported_diff/diff.patch -t /path/to/target
```
*Features automatic validation: If `diff.patch.sha256` is present alongside `diff.patch`, `goDirDiff` verifies cryptographic integrity before modifying any files in the target directory.*

#### Left / Dir1 Application (`-d lr`)
Brings in **left** (or top) changes that belong to `dir1` into the target directory:

```bash
# Bring dir1 changes into target
./goDirDiff -a -d lr -p exported_diff/diff.patch -t /path/to/target
```

---

### 7. Testing with Sample Directories & Pre-Generated Diff Files
The repository includes sample directory trees (`examples/dir_v1` and `examples/dir_v2`) and pre-generated Git-diff style reference files to illustrate how directory diffs look:

- `examples/sample_folder_diff.diff`: Standard Git unified diff format output.
- `examples/sample_folder_diff.patch`: Patch file ready for application.
- `examples/sample_folder_diff.txt`: Demonstration text file format.

#### Viewing the Sample Diff:
```bash
cat examples/sample_folder_diff.diff
# or
cat examples/sample_folder_diff.txt
```

#### Running & Testing with the Sample Directories:
```bash
# 1. Compare sample directories (emits diff to stdout)
./goDirDiff examples/dir_v1 examples/dir_v2

# 2. Export patch with companion checksum
./goDirDiff -e examples/dir_v1 examples/dir_v2

# 3. Test patch application in temporary directory
TEMP_DIR=$(mktemp -d)
cp -r examples/dir_v1/* "$TEMP_DIR"
./goDirDiff -a -p examples/sample_folder_diff.patch -t "$TEMP_DIR"
./goDirDiff "$TEMP_DIR" examples/dir_v2 # returns 0 (identical)
rm -rf "$TEMP_DIR"
```
