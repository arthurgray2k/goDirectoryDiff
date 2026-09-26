# USAGE: goDirDiff

## Basic Usage
```bash
./goDirDiff <directory_a> <directory_b>
```

## Options
| Flag | Description | Default |
|---|---|---|
| `-c`, `--context` | Number of context lines to display | `3` |
| `-f`, `--filter` | Filter diff by specific file or folder path (ignored if invalid) | `""` |
| `-e`, `--export` | Export diff to default directory (`exported_diff/`) | `false` |
| `-o`, `--export-path` | Specify custom export destination path or directory | `""` |
| `-a`, `--apply` | Apply/patch diff to target directory or between directories | `false` |
| `-p`, `--patch` | Path to patch file to apply | `exported_diff/diff.patch` |
| `-d`, `--direction` | Patch direction: `lr` (l->r, forward) or `rl` (r->l, reverse) | `lr` |
| `-t`, `--target` | Target directory to apply patch to | `""` |
| `--color` | Enable colored terminal output | `false` |
| `-h`, `--help` | Show command usage | |

## Exit Codes
- `0`: Directories are identical (no differences).
- `1`: Differences were found and printed to stdout.
- `2`: Error occurred (e.g. invalid arguments, directory inaccessible, export error).

## Exporting Diffs & SHA-256 Companion Files

### Export to Default Directory (`exported_diff/`)
Use `-e` or `--export` to save the unified diff to `exported_diff/diff.patch`:

```bash
./goDirDiff -e dir_v1 dir_v2
```

This generates:
- `exported_diff/diff.patch` (diff output)
- `exported_diff/diff.patch.sha256` (companion SHA-256 checksum)

### Export to Custom File or Directory Path
Use `-o` or `--export-path` to save the diff to a specified location:

```bash
# Export to a custom file
./goDirDiff -o output/release.patch dir_v1 dir_v2

# Export to a directory (diff.patch and diff.patch.sha256 written inside)
./goDirDiff -o /var/patches/ dir_v1 dir_v2
```

### Verifying SHA-256 Checksums
The companion checksum file uses standard GNU `sha256sum` format:

```bash
cd exported_diff
sha256sum -c diff.patch.sha256
# diff.patch: OK
```

## Filtering Diffs by File or Folder
You can scope the diff to a specific file or subfolder using `-f` / `--filter`:

```bash
./goDirDiff -f config.json examples/dir_v1 examples/dir_v2
```
*Note: If the provided filter path is invalid or matches nothing, the filter is automatically ignored and the full directory comparison is shown.*

## Demonstrating with Sample Files

The repository includes sample directories `examples/dir_v1` and `examples/dir_v2` to demonstrate how folder diffs work:

```bash
./goDirDiff examples/dir_v1 examples/dir_v2
```

## Applying Diffs & Patching (Bidirectional)

`goDirDiff` provides a built-in patch application engine supporting forward (`lr`, `l->r`) and reverse (`rl`, `r->l`) modifications.

### 1. Applying a Patch File (Forward: `l->r`)
Applies changes from a generated patch file to a target directory:

```bash
# Apply diff.patch to target directory (default direction is lr)
./goDirDiff -a -p exported_diff/diff.patch -t /path/to/target_dir

# Explicitly specifying forward direction
./goDirDiff -a -d lr -p exported_diff/diff.patch -t /path/to/target_dir
```
*Note: If a companion `<patch>.sha256` checksum exists in the same directory, `goDirDiff` automatically verifies cryptographic integrity before applying.*

### 2. Applying a Patch File in Reverse (`r->l`)
Reverses changes from a patch file (restoring deleted files, reverting modified lines, and removing added files):

```bash
./goDirDiff -a -d rl -p exported_diff/diff.patch -t /path/to/target_dir
```

### 3. Direct Synchronization Between Two Directories
Directly applies changes between two directory trees without saving an intermediate patch file:

```bash
# Updates dir_a to match dir_b (forward: lr)
./goDirDiff -a examples/dir_v1 examples/dir_v2

# Updates dir_b to match dir_a (reverse: rl)
./goDirDiff -a -d rl examples/dir_v1 examples/dir_v2
```


