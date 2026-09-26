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
	"strconv"
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

// PatchDirection specifies the direction to apply a patch.
type PatchDirection string

const (
	DirectionRL PatchDirection = "rl" // right: bring in right (dir2/bottom) changes to target directory
	DirectionLR PatchDirection = "lr" // left: bring in left (dir1/top) changes to target directory
)

// ParseDirection normalizes and parses a direction string.
func ParseDirection(s string) (PatchDirection, error) {
	norm := strings.ToLower(strings.TrimSpace(s))
	switch norm {
	case "rl", "r->l", "r→l", "r2l", "right", "r", "bottom":
		return DirectionRL, nil
	case "lr", "l->r", "l→r", "l2r", "left", "l", "top":
		return DirectionLR, nil
	default:
		return "", fmt.Errorf("invalid direction %q: must be 'rl' (right/dir2 changes) or 'lr' (left/dir1 changes)", s)
	}
}

// PatchLine represents a single line inside a unified diff hunk.
type PatchLine struct {
	Type rune   // ' ' (context), '-' (deletion), '+' (insertion)
	Text string // content of the line without prefix
}

// Hunk represents a single unified diff hunk.
type Hunk struct {
	OldStart int
	OldLen   int
	NewStart int
	NewLen   int
	Lines    []PatchLine
}

// FilePatch represents all changes for a single file in a diff.
type FilePatch struct {
	OldPath   string
	NewPath   string
	IsNew     bool
	IsDeleted bool
	IsBinary  bool
	Hunks     []Hunk
}

// ApplyResult contains the files modified, added, deleted, or skipped.
type ApplyResult struct {
	Modified []string
	Added    []string
	Deleted  []string
	Skipped  []string
}

func parseHunkHeader(line string) (oldStart, oldLen, newStart, newLen int, err error) {
	at1 := strings.Index(line, "@@")
	if at1 == -1 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: missing opening @@")
	}
	rest := line[at1+2:]
	at2 := strings.Index(rest, "@@")
	if at2 == -1 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: missing closing @@")
	}
	header := strings.TrimSpace(rest[:at2])
	parts := strings.Fields(header)
	if len(parts) < 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header format: %q", line)
	}

	pOld := strings.TrimPrefix(parts[0], "-")
	if strings.Contains(pOld, ",") {
		sub := strings.SplitN(pOld, ",", 2)
		oldStart, err = strconv.Atoi(sub[0])
		if err != nil {
			return 0, 0, 0, 0, err
		}
		oldLen, err = strconv.Atoi(sub[1])
		if err != nil {
			return 0, 0, 0, 0, err
		}
	} else {
		oldStart, err = strconv.Atoi(pOld)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		oldLen = 1
	}

	pNew := strings.TrimPrefix(parts[1], "+")
	if strings.Contains(pNew, ",") {
		sub := strings.SplitN(pNew, ",", 2)
		newStart, err = strconv.Atoi(sub[0])
		if err != nil {
			return 0, 0, 0, 0, err
		}
		newLen, err = strconv.Atoi(sub[1])
		if err != nil {
			return 0, 0, 0, 0, err
		}
	} else {
		newStart, err = strconv.Atoi(pNew)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		newLen = 1
	}

	return oldStart, oldLen, newStart, newLen, nil
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' || s[i] == 0x1B {
			inEsc = true
			continue
		}
		if inEsc {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// ParsePatch parses git unified diff content into FilePatch structures.
func ParsePatch(diffContent string) ([]FilePatch, error) {
	cleanContent := stripANSI(diffContent)
	rawLines := strings.Split(cleanContent, "\n")

	var patches []FilePatch
	var curPatch *FilePatch
	var curHunk *Hunk

	flushHunk := func() {
		if curPatch != nil && curHunk != nil {
			curPatch.Hunks = append(curPatch.Hunks, *curHunk)
			curHunk = nil
		}
	}

	flushPatch := func() {
		flushHunk()
		if curPatch != nil {
			patches = append(patches, *curPatch)
			curPatch = nil
		}
	}

	for _, line := range rawLines {
		if strings.HasPrefix(line, "diff --git ") {
			flushPatch()
			parts := strings.Fields(line)
			var oldP, newP string
			if len(parts) >= 4 {
				oldP = strings.TrimPrefix(parts[2], "a/")
				newP = strings.TrimPrefix(parts[3], "b/")
			}
			curPatch = &FilePatch{
				OldPath: oldP,
				NewPath: newP,
			}
			continue
		}

		if curPatch == nil {
			if strings.HasPrefix(line, "--- ") {
				curPatch = &FilePatch{}
			} else {
				continue
			}
		}

		if line == "new file mode 100644" {
			curPatch.IsNew = true
			continue
		}
		if line == "deleted file mode 100644" {
			curPatch.IsDeleted = true
			continue
		}
		if strings.HasPrefix(line, "Binary files ") {
			curPatch.IsBinary = true
			continue
		}
		if strings.HasPrefix(line, "--- ") {
			p := strings.TrimSpace(line[4:])
			if p == "/dev/null" {
				curPatch.IsNew = true
				curPatch.OldPath = "/dev/null"
			} else {
				curPatch.OldPath = strings.TrimPrefix(p, "a/")
			}
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			p := strings.TrimSpace(line[4:])
			if p == "/dev/null" {
				curPatch.IsDeleted = true
				curPatch.NewPath = "/dev/null"
			} else {
				curPatch.NewPath = strings.TrimPrefix(p, "b/")
			}
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			flushHunk()
			oldS, oldL, newS, newL, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			curHunk = &Hunk{
				OldStart: oldS,
				OldLen:   oldL,
				NewStart: newS,
				NewLen:   newL,
			}
			continue
		}

		if curHunk != nil {
			if len(line) == 0 {
				flushHunk()
				continue
			}
			prefix := line[0]
			switch prefix {
			case ' ':
				curHunk.Lines = append(curHunk.Lines, PatchLine{Type: ' ', Text: line[1:]})
			case '-':
				curHunk.Lines = append(curHunk.Lines, PatchLine{Type: '-', Text: line[1:]})
			case '+':
				curHunk.Lines = append(curHunk.Lines, PatchLine{Type: '+', Text: line[1:]})
			case '\\':
				continue
			default:
				flushHunk()
			}
		}
	}

	flushPatch()
	return patches, nil
}

func applyHunks(origLines []string, hunks []Hunk, dir PatchDirection) ([]string, error) {
	var result []string
	srcIdx := 0

	for _, hunk := range hunks {
		targetStart := 0
		if dir == DirectionRL {
			// rl: bring in right (dir2) changes. Matches baseline at OldStart
			targetStart = hunk.OldStart - 1
		} else {
			// lr: bring in left (dir1) changes. Matches baseline at NewStart
			targetStart = hunk.NewStart - 1
		}

		if targetStart < 0 {
			targetStart = 0
		}

		// Copy unmodified lines leading up to hunk
		if srcIdx < targetStart {
			if targetStart > len(origLines) {
				targetStart = len(origLines)
			}
			result = append(result, origLines[srcIdx:targetStart]...)
			srcIdx = targetStart
		}

		// Apply hunk lines
		for _, pl := range hunk.Lines {
			switch dir {
			case DirectionRL:
				// Right (dir2) changes:
				switch pl.Type {
				case ' ':
					if srcIdx < len(origLines) {
						result = append(result, origLines[srcIdx])
						srcIdx++
					} else {
						result = append(result, pl.Text)
					}
				case '-':
					// Remove left lines
					if srcIdx < len(origLines) {
						srcIdx++
					}
				case '+':
					// Insert right lines
					result = append(result, pl.Text)
				}
			case DirectionLR:
				// Left (dir1) changes:
				switch pl.Type {
				case ' ':
					if srcIdx < len(origLines) {
						result = append(result, origLines[srcIdx])
						srcIdx++
					} else {
						result = append(result, pl.Text)
					}
				case '+':
					// Remove right lines
					if srcIdx < len(origLines) {
						srcIdx++
					}
				case '-':
					// Restore left lines
					result = append(result, pl.Text)
				}
			}
		}
	}

	// Copy remaining lines after last hunk
	if srcIdx < len(origLines) {
		result = append(result, origLines[srcIdx:]...)
	}

	return result, nil
}

// ApplyPatch applies parsed file patches to targetDir in the specified direction.
func ApplyPatch(patches []FilePatch, targetDir string, dir PatchDirection) (*ApplyResult, error) {
	res := &ApplyResult{}

	for _, fp := range patches {
		if fp.IsBinary {
			res.Skipped = append(res.Skipped, fmt.Sprintf("binary: %s", fp.NewPath))
			continue
		}

		if dir == DirectionRL {
			// Bring in right (dir2) changes
			if fp.IsNew {
				filePath := filepath.Join(targetDir, fp.NewPath)
				if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
					return nil, fmt.Errorf("failed to create directory for %s: %w", fp.NewPath, err)
				}
				lines, err := applyHunks([]string{}, fp.Hunks, DirectionRL)
				if err != nil {
					return nil, fmt.Errorf("failed applying hunks to new file %s: %w", fp.NewPath, err)
				}
				var content string
				if len(lines) > 0 {
					content = strings.Join(lines, "\n") + "\n"
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					return nil, fmt.Errorf("failed to write new file %s: %w", filePath, err)
				}
				res.Added = append(res.Added, fp.NewPath)
			} else if fp.IsDeleted {
				filePath := filepath.Join(targetDir, fp.OldPath)
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					return nil, fmt.Errorf("failed to remove deleted file %s: %w", filePath, err)
				}
				res.Deleted = append(res.Deleted, fp.OldPath)
			} else {
				relPath := fp.NewPath
				if relPath == "" {
					relPath = fp.OldPath
				}
				filePath := filepath.Join(targetDir, relPath)
				contentBytes, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read file to patch %s: %w", filePath, err)
				}
				origLines := splitLines(string(contentBytes))
				newLines, err := applyHunks(origLines, fp.Hunks, DirectionRL)
				if err != nil {
					return nil, fmt.Errorf("failed applying hunks to %s: %w", relPath, err)
				}
				var newContent string
				if len(newLines) > 0 {
					newContent = strings.Join(newLines, "\n") + "\n"
				}
				if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
					return nil, fmt.Errorf("failed to write patched file %s: %w", filePath, err)
				}
				res.Modified = append(res.Modified, relPath)
			}
		} else { // DirectionLR: bring in left (dir1) changes
			if fp.IsNew {
				filePath := filepath.Join(targetDir, fp.NewPath)
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					return nil, fmt.Errorf("failed to remove file %s in left patch: %w", filePath, err)
				}
				res.Deleted = append(res.Deleted, fp.NewPath)
			} else if fp.IsDeleted {
				filePath := filepath.Join(targetDir, fp.OldPath)
				if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
					return nil, fmt.Errorf("failed to create directory for %s: %w", fp.OldPath, err)
				}
				lines, err := applyHunks([]string{}, fp.Hunks, DirectionLR)
				if err != nil {
					return nil, fmt.Errorf("failed applying hunks to restore file %s: %w", fp.OldPath, err)
				}
				var content string
				if len(lines) > 0 {
					content = strings.Join(lines, "\n") + "\n"
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					return nil, fmt.Errorf("failed to restore file %s: %w", filePath, err)
				}
				res.Added = append(res.Added, fp.OldPath)
			} else {
				relPath := fp.OldPath
				if relPath == "" {
					relPath = fp.NewPath
				}
				filePath := filepath.Join(targetDir, relPath)
				contentBytes, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read file to reverse patch %s: %w", filePath, err)
				}
				origLines := splitLines(string(contentBytes))
				newLines, err := applyHunks(origLines, fp.Hunks, DirectionLR)
				if err != nil {
					return nil, fmt.Errorf("failed applying reverse hunks to %s: %w", relPath, err)
				}
				var newContent string
				if len(newLines) > 0 {
					newContent = strings.Join(newLines, "\n") + "\n"
				}
				if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
					return nil, fmt.Errorf("failed to write reverse patched file %s: %w", filePath, err)
				}
				res.Modified = append(res.Modified, relPath)
			}
		}
	}

	return res, nil
}

// ApplyDiffFile reads a diff file from patchPath (verifying companion .sha256 if present)
// and applies it to targetDir in the specified direction.
func ApplyDiffFile(patchPath, targetDir string, dir PatchDirection) (*ApplyResult, error) {
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patch file %q: %w", patchPath, err)
	}

	hashPath := patchPath + ".sha256"
	if hashData, err := os.ReadFile(hashPath); err == nil {
		actualHash := ComputeSHA256(data)
		hashFields := strings.Fields(string(hashData))
		if len(hashFields) > 0 {
			expectedHash := hashFields[0]
			if !strings.EqualFold(actualHash, expectedHash) {
				return nil, fmt.Errorf("checksum mismatch for patch %q: expected %s, got %s", patchPath, expectedHash, actualHash)
			}
		}
	}

	patches, err := ParsePatch(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse patch file %q: %w", patchPath, err)
	}

	return ApplyPatch(patches, targetDir, dir)
}

// ApplyBetweenDirectories compares dirA and dirB and synchronizes changes in direction dir.
// If dir == DirectionLR, dirA is updated to match dirB.
// If dir == DirectionRL, dirB is updated to match dirA.
func ApplyBetweenDirectories(dirA, dirB string, opts Options, dir PatchDirection) (*ApplyResult, error) {
	diffOutput, hasDiff, err := CompareDirectories(dirA, dirB, opts)
	if err != nil {
		return nil, err
	}
	if !hasDiff {
		return &ApplyResult{}, nil
	}

	patches, err := ParsePatch(diffOutput)
	if err != nil {
		return nil, err
	}

	var targetDir string
	if dir == DirectionRL {
		targetDir = dirA
	} else {
		targetDir = dirB
	}

	res, err := ApplyPatch(patches, targetDir, dir)
	if err != nil {
		return nil, err
	}

	for _, fp := range patches {
		if fp.IsBinary {
			var srcPath, dstPath string
			relPath := fp.NewPath
			if relPath == "" {
				relPath = fp.OldPath
			}
			if dir == DirectionRL {
				srcPath = filepath.Join(dirB, relPath)
				dstPath = filepath.Join(dirA, relPath)
			} else {
				srcPath = filepath.Join(dirA, relPath)
				dstPath = filepath.Join(dirB, relPath)
			}

			if srcData, err := os.ReadFile(srcPath); err == nil {
				if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err == nil {
					if err := os.WriteFile(dstPath, srcData, 0644); err == nil {
						res.Modified = append(res.Modified, relPath)
					}
				}
			} else if os.IsNotExist(err) {
				os.Remove(dstPath)
				res.Deleted = append(res.Deleted, relPath)
			}
		}
	}

	return res, nil
}
