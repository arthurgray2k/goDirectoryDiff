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

func TestParseDirection(t *testing.T) {
	lrTests := []string{"lr", "LR", "l->r", "L->R", "l→r", "l2r", "forward", "fwd", "left-to-right"}
	for _, s := range lrTests {
		dir, err := ParseDirection(s)
		if err != nil || dir != DirectionLR {
			t.Errorf("expected %s -> DirectionLR, got dir=%v, err=%v", s, dir, err)
		}
	}

	rlTests := []string{"rl", "RL", "r->l", "R->L", "r→l", "r2l", "reverse", "rev", "right-to-left"}
	for _, s := range rlTests {
		dir, err := ParseDirection(s)
		if err != nil || dir != DirectionRL {
			t.Errorf("expected %s -> DirectionRL, got dir=%v, err=%v", s, dir, err)
		}
	}

	invalidTests := []string{"", "invalid", "up", "down", "random"}
	for _, s := range invalidTests {
		_, err := ParseDirection(s)
		if err == nil {
			t.Errorf("expected error for invalid direction %q", s)
		}
	}
}

func TestStripANSI(t *testing.T) {
	input := "\033[31mred\033[0m normal \033[32mgreen\033[0m"
	expected := "red normal green"
	actual := stripANSI(input)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestParseHunkHeader(t *testing.T) {
	// Standard
	osVal, ol, ns, nl, err := parseHunkHeader("@@ -10,4 +20,6 @@ optional context")
	if err != nil || osVal != 10 || ol != 4 || ns != 20 || nl != 6 {
		t.Errorf("unexpected parse result: %d,%d, %d,%d, err=%v", osVal, ol, ns, nl, err)
	}

	// Single line
	osVal, ol, ns, nl, err = parseHunkHeader("@@ -1 +1 @@")
	if err != nil || osVal != 1 || ol != 1 || ns != 1 || nl != 1 {
		t.Errorf("single line hunk parse failed: %d,%d, %d,%d, err=%v", osVal, ol, ns, nl, err)
	}

	// Errors
	if _, _, _, _, err := parseHunkHeader("invalid line"); err == nil {
		t.Errorf("expected error for missing @@")
	}
	if _, _, _, _, err := parseHunkHeader("@@ unclosed"); err == nil {
		t.Errorf("expected error for unclosed @@")
	}
	if _, _, _, _, err := parseHunkHeader("@@ @@"); err == nil {
		t.Errorf("expected error for empty hunk header")
	}
	if _, _, _, _, err := parseHunkHeader("@@ -abc +def @@"); err == nil {
		t.Errorf("expected error for non-numeric hunk header")
	}
	if _, _, _, _, err := parseHunkHeader("@@ -1,abc +def @@"); err == nil {
		t.Errorf("expected error for non-numeric hunk header len")
	}
	if _, _, _, _, err := parseHunkHeader("@@ -1 +abc @@"); err == nil {
		t.Errorf("expected error for non-numeric new start")
	}
	if _, _, _, _, err := parseHunkHeader("@@ -1 +1,abc @@"); err == nil {
		t.Errorf("expected error for non-numeric new len")
	}
}

func TestParsePatch_And_ApplyPatch_Bidirectional(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	// In dirA: modified.txt (old), deleted.txt
	os.WriteFile(filepath.Join(dirA, "modified.txt"), []byte("line 1\nline 2 old\nline 3\n"), 0644)
	os.WriteFile(filepath.Join(dirA, "deleted.txt"), []byte("delete me\n"), 0644)

	// In dirB: modified.txt (new), added.txt
	os.WriteFile(filepath.Join(dirB, "modified.txt"), []byte("line 1\nline 2 new\nline 3\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "added.txt"), []byte("brand new file\n"), 0644)

	diffOut, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareDirectories failed: %v", err)
	}
	if !hasDiff {
		t.Fatalf("expected differences between dirA and dirB")
	}

	patches, err := ParsePatch(diffOut)
	if err != nil {
		t.Fatalf("ParsePatch failed: %v", err)
	}

	// 1. Forward apply (DirectionLR): patch a copy of dirA -> should match dirB
	targetLR := t.TempDir()
	os.WriteFile(filepath.Join(targetLR, "modified.txt"), []byte("line 1\nline 2 old\nline 3\n"), 0644)
	os.WriteFile(filepath.Join(targetLR, "deleted.txt"), []byte("delete me\n"), 0644)

	resLR, err := ApplyPatch(patches, targetLR, DirectionLR)
	if err != nil {
		t.Fatalf("ApplyPatch LR failed: %v", err)
	}
	if len(resLR.Modified) != 1 || len(resLR.Added) != 1 || len(resLR.Deleted) != 1 {
		t.Errorf("unexpected LR apply result: %+v", resLR)
	}

	diffAfterLR, hasDiffLR, err := CompareDirectories(targetLR, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareDirectories after LR failed: %v", err)
	}
	if hasDiffLR {
		t.Errorf("targetLR does not match dirB: %s", diffAfterLR)
	}

	// 2. Reverse apply (DirectionRL): patch a copy of dirB -> should match dirA
	targetRL := t.TempDir()
	os.WriteFile(filepath.Join(targetRL, "modified.txt"), []byte("line 1\nline 2 new\nline 3\n"), 0644)
	os.WriteFile(filepath.Join(targetRL, "added.txt"), []byte("brand new file\n"), 0644)

	resRL, err := ApplyPatch(patches, targetRL, DirectionRL)
	if err != nil {
		t.Fatalf("ApplyPatch RL failed: %v", err)
	}
	if len(resRL.Modified) != 1 || len(resRL.Added) != 1 || len(resRL.Deleted) != 1 {
		t.Errorf("unexpected RL apply result: %+v", resRL)
	}

	diffAfterRL, hasDiffRL, err := CompareDirectories(targetRL, dirA, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareDirectories after RL failed: %v", err)
	}
	if hasDiffRL {
		t.Errorf("targetRL does not match dirA: %s", diffAfterRL)
	}
}

func TestApplyDiffFile_Success_And_ChecksumMismatch(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	os.WriteFile(filepath.Join(dirA, "file.txt"), []byte("version 1\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "file.txt"), []byte("version 2\n"), 0644)

	diffOut, _, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareDirectories failed: %v", err)
	}

	tmpDir := t.TempDir()
	patchFile := filepath.Join(tmpDir, "changes.patch")
	exportRes, err := ExportDiff(diffOut, patchFile)
	if err != nil {
		t.Fatalf("ExportDiff failed: %v", err)
	}

	targetDir := t.TempDir()
	os.WriteFile(filepath.Join(targetDir, "file.txt"), []byte("version 1\n"), 0644)

	// Apply valid patch
	applyRes, err := ApplyDiffFile(exportRes.DiffPath, targetDir, DirectionLR)
	if err != nil {
		t.Fatalf("ApplyDiffFile failed: %v", err)
	}
	if len(applyRes.Modified) != 1 {
		t.Errorf("expected 1 modified file, got: %+v", applyRes)
	}

	updatedContent, _ := os.ReadFile(filepath.Join(targetDir, "file.txt"))
	if string(updatedContent) != "version 2\n" {
		t.Errorf("content not updated: %q", string(updatedContent))
	}

	// Corrupt patch file to test checksum verification
	if err := os.WriteFile(exportRes.DiffPath, []byte("tampered content"), 0644); err != nil {
		t.Fatalf("failed to tamper patch file: %v", err)
	}
	_, err = ApplyDiffFile(exportRes.DiffPath, targetDir, DirectionLR)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}

	// Non-existent patch file
	_, err = ApplyDiffFile(filepath.Join(tmpDir, "missing.patch"), targetDir, DirectionLR)
	if err == nil {
		t.Errorf("expected error for non-existent patch file")
	}
}

func TestApplyBetweenDirectories(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	// Text files
	os.WriteFile(filepath.Join(dirA, "text.txt"), []byte("original text\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "text.txt"), []byte("updated text\n"), 0644)

	// Binary files
	binA := []byte{0x00, 0x01, 0x02}
	binB := []byte{0x00, 0x01, 0x03}
	os.WriteFile(filepath.Join(dirA, "data.bin"), binA, 0644)
	os.WriteFile(filepath.Join(dirB, "data.bin"), binB, 0644)

	// Added binary file in dirB
	binExtra := []byte{0x00, 0xFF, 0xFE}
	os.WriteFile(filepath.Join(dirB, "extra.bin"), binExtra, 0644)

	// Synchronize dirA -> dirB using DirectionLR (updates dirA to match dirB)
	res, err := ApplyBetweenDirectories(dirA, dirB, DefaultOptions(), DirectionLR)
	if err != nil {
		t.Fatalf("ApplyBetweenDirectories LR failed: %v", err)
	}
	if len(res.Modified) == 0 {
		t.Errorf("expected modified files, got: %+v", res)
	}

	diffAfter, hasDiff, err := CompareDirectories(dirA, dirB, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareDirectories failed: %v", err)
	}
	if hasDiff {
		t.Errorf("expected dirA and dirB to be synchronized, diff: %s", diffAfter)
	}

	// Test identical directories
	resIdentical, err := ApplyBetweenDirectories(dirA, dirB, DefaultOptions(), DirectionLR)
	if err != nil {
		t.Fatalf("unexpected error for identical directories: %v", err)
	}
	if len(resIdentical.Modified) != 0 || len(resIdentical.Added) != 0 {
		t.Errorf("expected empty result for identical dirs, got: %+v", resIdentical)
	}

	// Test DirectionRL: update dirB to match a modified dirA
	os.WriteFile(filepath.Join(dirA, "text.txt"), []byte("text changed again in A\n"), 0644)
	resRL, err := ApplyBetweenDirectories(dirA, dirB, DefaultOptions(), DirectionRL)
	if err != nil {
		t.Fatalf("ApplyBetweenDirectories RL failed: %v", err)
	}
	if len(resRL.Modified) == 0 {
		t.Errorf("expected modified files in RL, got: %+v", resRL)
	}
	diffAfterRL, hasDiffRL, _ := CompareDirectories(dirA, dirB, DefaultOptions())
	if hasDiffRL {
		t.Errorf("expected dirA and dirB to be synchronized in RL, diff: %s", diffAfterRL)
	}
}

func TestApplyPatch_ErrorAndEdgeCases(t *testing.T) {
	// Attempt to patch a non-existent file
	targetDir := t.TempDir()
	patch := FilePatch{
		OldPath: "nonexistent.txt",
		NewPath: "nonexistent.txt",
		Hunks: []Hunk{
			{
				OldStart: 1,
				OldLen:   1,
				NewStart: 1,
				NewLen:   1,
				Lines: []PatchLine{
					{Type: '-', Text: "old"},
					{Type: '+', Text: "new"},
				},
			},
		},
	}

	_, err := ApplyPatch([]FilePatch{patch}, targetDir, DirectionLR)
	if err == nil {
		t.Errorf("expected error when patching non-existent file")
	}

	// Binary patch should be skipped
	binaryPatch := FilePatch{
		OldPath:  "binary.bin",
		NewPath:  "binary.bin",
		IsBinary: true,
	}
	res, err := ApplyPatch([]FilePatch{binaryPatch}, targetDir, DirectionLR)
	if err != nil {
		t.Fatalf("unexpected error on binary patch: %v", err)
	}
	if len(res.Skipped) != 1 {
		t.Errorf("expected binary patch to be skipped, got: %+v", res)
	}

	// Apply with empty patch list
	emptyRes, err := ApplyPatch([]FilePatch{}, targetDir, DirectionLR)
	if err != nil {
		t.Fatalf("unexpected error on empty patches: %v", err)
	}
	if len(emptyRes.Modified) != 0 {
		t.Errorf("expected empty result")
	}
}
