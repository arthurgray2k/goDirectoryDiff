# USAGE: goDirectoryDiff

## Basic Usage
```bash
godirectorydiff <directory_a> <directory_b>
```

## Options
| Flag | Description | Default |
|---|---|---|
| `-c`, `--context` | Number of context lines to display | `3` |
| `--color` | Enable colored terminal output | `false` |
| `-h`, `--help` | Show command usage | |

## Exit Codes
- `0`: Directories are identical (no differences).
- `1`: Differences were found and printed to stdout.
- `2`: Error occurred (e.g. invalid arguments, directory inaccessible).

## Examples

### 1. Compare two directories
```bash
./godirectorydiff /path/to/project_v1 /path/to/project_v2
```

### 2. Compare with color highlighting
```bash
./godirectorydiff --color ./dirA ./dirB
```

### 3. Compare with custom context lines
```bash
./godirectorydiff -c 5 ./dirA ./dirB
```

### 4. Redirecting diff to a patch file
```bash
./godirectorydiff ./dirA ./dirB > my_changes.patch
```
