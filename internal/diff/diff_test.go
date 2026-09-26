package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareDirectories_Identical(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	os.WriteFile(filepath.Join(dirA, "file1.txt"), []byte("hello world\nline 2\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "file1.txt"), []byte("hello world\nline 2\n"), 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasDiff {
		t.Errorf("expected no diff, got: %s", diffOut)
	}
	if diffOut != "" {
		t.Errorf("expected empty diff string, got: %s", diffOut)
	}
}

func TestCompareDirectories_Modified(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	os.WriteFile(filepath.Join(dirA, "file1.txt"), []byte("alpha\nbeta\ngamma\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "file1.txt"), []byte("alpha\nbeta updated\ngamma\n"), 0644)

	opts := Options{ContextLines: 1, Color: false}
	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected diff, but got none")
	}
	if !strings.Contains(diffOut, "diff --git a/file1.txt b/file1.txt") {
		t.Errorf("expected git diff header, got: %s", diffOut)
	}
	if !strings.Contains(diffOut, "-beta") || !strings.Contains(diffOut, "+beta updated") {
		t.Errorf("expected hunk lines, got: %s", diffOut)
	}
}

func TestCompareDirectories_Color(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	os.WriteFile(filepath.Join(dirA, "f.txt"), []byte("line1\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "f.txt"), []byte("line2\n"), 0644)

	opts := Options{ContextLines: 3, Color: true}
	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected diff, but got none")
	}
	if !strings.Contains(diffOut, "\033[31m-") || !strings.Contains(diffOut, "\033[32m+") {
		t.Errorf("expected colorized output, got: %s", diffOut)
	}
}

func TestCompareDirectories_AddedAndDeletedFiles(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	// In dirA only -> Deleted in dirB
	os.WriteFile(filepath.Join(dirA, "deleted.txt"), []byte("to be deleted\n"), 0644)
	// In dirB only -> Added in dirB
	os.WriteFile(filepath.Join(dirB, "added.txt"), []byte("new content\n"), 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected diff, but got none")
	}
	if !strings.Contains(diffOut, "new file mode 100644") {
		t.Errorf("expected new file header, got: %s", diffOut)
	}
	if !strings.Contains(diffOut, "deleted file mode 100644") {
		t.Errorf("expected deleted file header, got: %s", diffOut)
	}
}

func TestCompareDirectories_BinaryFiles(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	binA := []byte{0x00, 0x01, 0x02, 0x03}
	binB := []byte{0x00, 0x01, 0x02, 0x04}

	os.WriteFile(filepath.Join(dirA, "binary.bin"), binA, 0644)
	os.WriteFile(filepath.Join(dirB, "binary.bin"), binB, 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected binary diff, but got none")
	}
	if !strings.Contains(diffOut, "Binary files a/binary.bin and b/binary.bin differ") {
		t.Errorf("expected binary diff warning, got: %s", diffOut)
	}
}

func TestCompareDirectories_AddedBinaryFile(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	binB := []byte{0x00, 0xFF, 0xFE}
	os.WriteFile(filepath.Join(dirB, "added_binary.bin"), binB, 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected binary diff, got none")
	}
	if !strings.Contains(diffOut, "Binary files /dev/null and b/added_binary.bin differ") {
		t.Errorf("expected binary diff message, got: %s", diffOut)
	}
}

func TestCompareDirectories_DeletedBinaryFile(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	binA := []byte{0x00, 0xAA, 0xBB}
	os.WriteFile(filepath.Join(dirA, "deleted_binary.bin"), binA, 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected binary diff, got none")
	}
	if !strings.Contains(diffOut, "Binary files a/deleted_binary.bin and /dev/null differ") {
		t.Errorf("expected binary diff message, got: %s", diffOut)
	}
}

func TestCompareDirectories_InvalidDirs(t *testing.T) {
	_, _, err := CompareDirectories("nonexistent_path_a", "nonexistent_path_b", DefaultOptions())
	if err == nil {
		t.Errorf("expected error for nonexistent directory")
	}

	validDir := t.TempDir()
	filePath := filepath.Join(validDir, "file.txt")
	os.WriteFile(filePath, []byte("test"), 0644)

	_, _, err = CompareDirectories(filePath, validDir, DefaultOptions())
	if err == nil {
		t.Errorf("expected error when path is not a directory")
	}
}

func TestCompareDirectories_Filter(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	os.WriteFile(filepath.Join(dirA, "matched.txt"), []byte("v1\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "matched.txt"), []byte("v2\n"), 0644)

	os.WriteFile(filepath.Join(dirA, "ignored.txt"), []byte("v1\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "ignored.txt"), []byte("v2\n"), 0644)

	// Valid filter: should only show matched.txt
	opts := Options{ContextLines: 3, Filter: "matched.txt"}
	out, hasDiff, err := CompareDirectories(dirA, dirB, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiff {
		t.Errorf("expected diff for matched.txt")
	}
	if !strings.Contains(out, "matched.txt") {
		t.Errorf("expected output to contain matched.txt")
	}
	if strings.Contains(out, "ignored.txt") {
		t.Errorf("expected output NOT to contain ignored.txt")
	}

	// Invalid filter: should fall back to showing all diffs
	optsInvalid := Options{ContextLines: 3, Filter: "nonexistent/folder/path"}
	outInvalid, hasDiffInvalid, err := CompareDirectories(dirA, dirB, optsInvalid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDiffInvalid {
		t.Errorf("expected fallback diff")
	}
	if !strings.Contains(outInvalid, "matched.txt") || !strings.Contains(outInvalid, "ignored.txt") {
		t.Errorf("expected both files to be included when filter is invalid")
	}
}

func TestComputeSHA256(t *testing.T) {
	data := []byte("hello world\n")
	expected := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	actual := ComputeSHA256(data)
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestExportDiff_CustomFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "output", "my_changes.patch")
	sampleDiff := "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-old\n+new\n"

	res, err := ExportDiff(sampleDiff, targetFile)
	if err != nil {
		t.Fatalf("ExportDiff failed: %v", err)
	}

	if res.DiffPath != targetFile {
		t.Errorf("expected diff path %q, got %q", targetFile, res.DiffPath)
	}
	expectedHashPath := targetFile + ".sha256"
	if res.HashPath != expectedHashPath {
		t.Errorf("expected hash path %q, got %q", expectedHashPath, res.HashPath)
	}

	// Verify diff file content
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read exported diff: %v", err)
	}
	if string(content) != sampleDiff {
		t.Errorf("exported diff content mismatch: got %q, expected %q", string(content), sampleDiff)
	}

	// Verify hash file content
	hashContent, err := os.ReadFile(expectedHashPath)
	if err != nil {
		t.Fatalf("failed to read hash file: %v", err)
	}
	expectedHashFileContent := fmt.Sprintf("%s  my_changes.patch\n", res.SHA256)
	if string(hashContent) != expectedHashFileContent {
		t.Errorf("hash file mismatch: got %q, expected %q", string(hashContent), expectedHashFileContent)
	}
}

func TestExportDiff_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "export_dir")
	sampleDiff := "diff --git a/a b/b\n"

	res, err := ExportDiff(sampleDiff, outDir+"/")
	if err != nil {
		t.Fatalf("ExportDiff with directory path failed: %v", err)
	}

	expectedDiffPath := filepath.Join(outDir, "diff.patch")
	if res.DiffPath != expectedDiffPath {
		t.Errorf("expected %q, got %q", expectedDiffPath, res.DiffPath)
	}
	if _, err := os.Stat(res.HashPath); err != nil {
		t.Errorf("companion sha256 file does not exist: %v", err)
	}
}

func TestExportDiff_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}
	defer os.Chdir(originalWd)

	res, err := ExportDiff("sample diff", "")
	if err != nil {
		t.Fatalf("ExportDiff with default path failed: %v", err)
	}

	expectedDiffPath := filepath.Join("exported_diff", "diff.patch")
	if res.DiffPath != expectedDiffPath {
		t.Errorf("expected %q, got %q", expectedDiffPath, res.DiffPath)
	}
	if _, err := os.Stat(expectedDiffPath); err != nil {
		t.Errorf("default diff file was not created: %v", err)
	}
	if _, err := os.Stat(expectedDiffPath + ".sha256"); err != nil {
		t.Errorf("default hash file was not created: %v", err)
	}
}
