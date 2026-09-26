package diff

import (
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
