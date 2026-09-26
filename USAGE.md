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
| `--color` | Enable colored terminal output | `false` |
| `-h`, `--help` | Show command usage | |

## Exit Codes
- `0`: Directories are identical (no differences).
- `1`: Differences were found and printed to stdout.
- `2`: Error occurred (e.g. invalid arguments, directory inaccessible).

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
