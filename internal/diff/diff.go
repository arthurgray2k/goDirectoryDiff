package diff

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options holds configuration for comparing directories.
type Options struct {
	ContextLines int
	Color        bool
	Filter       string
}

// DefaultOptions returns standard comparison options.
func DefaultOptions() Options {
	return Options{
		ContextLines: 3,
		Color:        false,
		Filter:       "",
	}
}

// CompareDirectories recursively compares dirA and dirB and returns the diff output
// formatted similarly to git diff. Returns (diffOutput, hasDiff, error).
func CompareDirectories(dirA, dirB string, opts Options) (string, bool, error) {
	infoA, err := os.Stat(dirA)
	if err != nil {
		return "", false, fmt.Errorf("failed to access dirA %q: %w", dirA, err)
	}
	if !infoA.IsDir() {
		return "", false, fmt.Errorf("dirA %q is not a directory", dirA)
	}

	infoB, err := os.Stat(dirB)
	if err != nil {
		return "", false, fmt.Errorf("failed to access dirB %q: %w", dirB, err)
	}
	if !infoB.IsDir() {
		return "", false, fmt.Errorf("dirB %q is not a directory", dirB)
	}

	filesA, err := collectFiles(dirA)
	if err != nil {
		return "", false, err
	}

	filesB, err := collectFiles(dirB)
	if err != nil {
		return "", false, err
	}

	allRelPaths := unionKeys(filesA, filesB)

	// Apply filter if specified and matches valid paths; ignore if invalid
	filter := filepath.ToSlash(strings.TrimSpace(opts.Filter))
	if filter != "" {
		filterPrefix := strings.TrimSuffix(filter, "/") + "/"
		var filtered []string
		for _, relPath := range allRelPaths {
			if relPath == filter || strings.HasPrefix(relPath, filterPrefix) {
				filtered = append(filtered, relPath)
			}
		}
		if len(filtered) > 0 {
			allRelPaths = filtered
		}
		// If len(filtered) == 0, the filter path is invalid or has no matches, so ignore the filter as required
	}

	var sb strings.Builder
	hasDiff := false

	for _, relPath := range allRelPaths {
		_, inA := filesA[relPath]
		_, inB := filesB[relPath]

		pathA := filepath.Join(dirA, relPath)
		pathB := filepath.Join(dirB, relPath)

		if inA && inB {
			// File exists in both
			diffStr, diffFound, err := compareTwoFiles(pathA, pathB, relPath, opts)
			if err != nil {
				return "", false, err
			}
			if diffFound {
				hasDiff = true
				sb.WriteString(diffStr)
			}
		} else if inA && !inB {
			// File deleted in B
			diffStr, err := formatDeletedFile(pathA, relPath, opts)
			if err != nil {
				return "", false, err
			}
			hasDiff = true
			sb.WriteString(diffStr)
		} else if !inA && inB {
			// File added in B
			diffStr, err := formatAddedFile(pathB, relPath, opts)
			if err != nil {
				return "", false, err
			}
			hasDiff = true
			sb.WriteString(diffStr)
		}
	}

	return sb.String(), hasDiff, nil
}

func collectFiles(root string) (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes for git diff consistency
		files[filepath.ToSlash(rel)] = true
		return nil
	})
	return files, err
}

func unionKeys(a, b map[string]bool) []string {
	seen := make(map[string]bool)
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isBinary(data []byte) bool {
	checkLen := 8000
	if len(data) < checkLen {
		checkLen = len(data)
	}
	return bytes.ContainsRune(data[:checkLen], 0)
}

func compareTwoFiles(pathA, pathB, relPath string, opts Options) (string, bool, error) {
	contentA, err := os.ReadFile(pathA)
	if err != nil {
		return "", false, err
	}
	contentB, err := os.ReadFile(pathB)
	if err != nil {
		return "", false, err
	}

	if bytes.Equal(contentA, contentB) {
		return "", false, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", relPath, relPath))

	if isBinary(contentA) || isBinary(contentB) {
		sb.WriteString(fmt.Sprintf("Binary files a/%s and b/%s differ\n", relPath, relPath))
		return sb.String(), true, nil
	}

	linesA := splitLines(string(contentA))
	linesB := splitLines(string(contentB))

	unified := generateUnifiedDiff(linesA, linesB, "a/"+relPath, "b/"+relPath, opts.ContextLines, opts.Color)
	sb.WriteString(unified)
	return sb.String(), true, nil
}

func formatAddedFile(pathB, relPath string, opts Options) (string, error) {
	contentB, err := os.ReadFile(pathB)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", relPath, relPath))
	sb.WriteString("new file mode 100644\n")

	if isBinary(contentB) {
		sb.WriteString(fmt.Sprintf("Binary files /dev/null and b/%s differ\n", relPath))
		return sb.String(), nil
	}

	linesB := splitLines(string(contentB))
	unified := generateUnifiedDiff([]string{}, linesB, "/dev/null", "b/"+relPath, opts.ContextLines, opts.Color)
	sb.WriteString(unified)
	return sb.String(), nil
}

func formatDeletedFile(pathA, relPath string, opts Options) (string, error) {
	contentA, err := os.ReadFile(pathA)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", relPath, relPath))
	sb.WriteString("deleted file mode 100644\n")

	if isBinary(contentA) {
		sb.WriteString(fmt.Sprintf("Binary files a/%s and /dev/null differ\n", relPath))
		return sb.String(), nil
	}

	linesA := splitLines(string(contentA))
	unified := generateUnifiedDiff(linesA, []string{}, "a/"+relPath, "/dev/null", opts.ContextLines, opts.Color)
	sb.WriteString(unified)
	return sb.String(), nil
}

func splitLines(s string) []string {
	if len(s) == 0 {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp int

const (
	opEqual diffOp = iota
	opDelete
	opInsert
)

type diffItem struct {
	op   diffOp
	text string
}

func computeDiff(a, b []string) []diffItem {
	n := len(a)
	m := len(b)

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	var items []diffItem
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			items = append(items, diffItem{op: opEqual, text: a[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			items = append(items, diffItem{op: opInsert, text: b[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]) {
			items = append(items, diffItem{op: opDelete, text: a[i-1]})
			i--
		}
	}

	for k := 0; k < len(items)/2; k++ {
		items[k], items[len(items)-1-k] = items[len(items)-1-k], items[k]
	}

	return items
}

func generateUnifiedDiff(a, b []string, labelA, labelB string, contextLines int, color bool) string {
	if contextLines < 0 {
		contextLines = 3
	}
	diffItems := computeDiff(a, b)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n", labelA))
	sb.WriteString(fmt.Sprintf("+++ %s\n", labelB))

	type hunkLine struct {
		op   diffOp
		text string
	}

	var currentHunk []hunkLine
	startA, lenA := 1, 0
	startB, lenB := 1, 0
	inHunk := false
	equalCount := 0

	lineA := 1
	lineB := 1

	flushHunk := func() {
		if len(currentHunk) == 0 {
			return
		}
		// Trim trailing equals beyond contextLines
		for len(currentHunk) > 0 && currentHunk[len(currentHunk)-1].op == opEqual && equalCount > contextLines {
			last := currentHunk[len(currentHunk)-1]
			if last.op == opEqual {
				lenA--
				lenB--
			}
			currentHunk = currentHunk[:len(currentHunk)-1]
			equalCount--
		}

		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startA, lenA, startB, lenB))
		for _, hl := range currentHunk {
			switch hl.op {
			case opEqual:
				sb.WriteString(" " + hl.text + "\n")
			case opDelete:
				if color {
					sb.WriteString("\033[31m-" + hl.text + "\033[0m\n")
				} else {
					sb.WriteString("-" + hl.text + "\n")
				}
			case opInsert:
				if color {
					sb.WriteString("\033[32m+" + hl.text + "\033[0m\n")
				} else {
					sb.WriteString("+" + hl.text + "\n")
				}
			}
		}
		currentHunk = nil
		inHunk = false
	}

	for idx, item := range diffItems {
		if item.op != opEqual {
			if !inHunk {
				inHunk = true
				startA = lineA
				startB = lineB
				lenA = 0
				lenB = 0
				// Prepend leading context lines
				leadingStart := idx - contextLines
				if leadingStart < 0 {
					leadingStart = 0
				}
				for k := leadingStart; k < idx; k++ {
					if diffItems[k].op == opEqual {
						currentHunk = append(currentHunk, hunkLine{op: opEqual, text: diffItems[k].text})
						lenA++
						lenB++
						startA--
						startB--
					}
				}
				if startA < 1 {
					startA = 1
				}
				if startB < 1 {
					startB = 1
				}
			}
			currentHunk = append(currentHunk, hunkLine{op: item.op, text: item.text})
			if item.op == opDelete {
				lenA++
				lineA++
			} else {
				lenB++
				lineB++
			}
			equalCount = 0
		} else {
			if inHunk {
				currentHunk = append(currentHunk, hunkLine{op: opEqual, text: item.text})
				lenA++
				lenB++
				equalCount++
				if equalCount >= contextLines*2 {
					flushHunk()
				}
			}
			lineA++
			lineB++
		}
	}

	if inHunk {
		flushHunk()
	}

	return sb.String()
}

// ExportResult holds the file paths and hash of an exported diff.
type ExportResult struct {
	DiffPath string
	HashPath string
	SHA256   string
}

// ComputeSHA256 returns the hex-encoded SHA-256 hash of the given data.
func ComputeSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ExportDiff exports diffContent to targetPath, writes a companion SHA-256 checksum
// file in the same directory, and returns an ExportResult.
// If targetPath is empty, it defaults to "exported_diff/diff.patch".
// If targetPath is a directory or ends with a separator, "diff.patch" is written inside it.
func ExportDiff(diffContent string, targetPath string) (*ExportResult, error) {
	cleanPath := strings.TrimSpace(targetPath)
	if cleanPath == "" {
		cleanPath = filepath.Join("exported_diff", "diff.patch")
	} else {
		cleanPath = filepath.Clean(cleanPath)
		// Check if cleanPath is an existing directory
		if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
			cleanPath = filepath.Join(cleanPath, "diff.patch")
		} else if strings.HasSuffix(targetPath, "/") || strings.HasSuffix(targetPath, string(filepath.Separator)) {
			// Explicit trailing slash implies directory
			cleanPath = filepath.Join(cleanPath, "diff.patch")
		}
	}

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	contentBytes := []byte(diffContent)
	if err := os.WriteFile(cleanPath, contentBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write diff to %q: %w", cleanPath, err)
	}

	hashHex := ComputeSHA256(contentBytes)
	hashPath := cleanPath + ".sha256"
	hashContent := fmt.Sprintf("%s  %s\n", hashHex, filepath.Base(cleanPath))

	if err := os.WriteFile(hashPath, []byte(hashContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write hash file to %q: %w", hashPath, err)
	}

	return &ExportResult{
		DiffPath: cleanPath,
		HashPath: hashPath,
		SHA256:   hashHex,
	}, nil
}
