package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"goDirectoryDiff/internal/diff"
)

func main() {
	contextFlag := flag.Int("context", 3, "Number of context lines to display")
	flag.IntVar(contextFlag, "c", 3, "Number of context lines (shorthand)")
	colorFlag := flag.Bool("color", false, "Enable colorized diff output")
	filterFlag := flag.String("filter", "", "Filter diff by specific file or folder path (ignored if invalid)")
	flag.StringVar(filterFlag, "f", "", "Filter diff by specific file or folder path (shorthand)")
	exportFlag := flag.Bool("export", false, "Export diff to default directory (exported_diff/)")
	flag.BoolVar(exportFlag, "e", false, "Export diff to default directory (shorthand)")
	exportPathFlag := flag.String("export-path", "", "Path to export diff file or directory (default: exported_diff/)")
	flag.StringVar(exportPathFlag, "o", "", "Path to export diff file or directory (shorthand)")

	applyFlag := flag.Bool("apply", false, "Apply/patch exported diff to target directory")
	flag.BoolVar(applyFlag, "a", false, "Apply/patch diff (shorthand)")
	patchPathFlag := flag.String("patch", "", "Path to patch file to apply (default: exported_diff/diff.patch)")
	flag.StringVar(patchPathFlag, "p", "", "Path to patch file (shorthand)")
	directionFlag := flag.String("direction", "rl", "Patch direction: 'rl' (right/dir2 changes) or 'lr' (left/dir1 changes)")
	flag.StringVar(directionFlag, "d", "rl", "Patch direction (shorthand)")
	targetFlag := flag.String("target", "", "Target directory to apply patch to")
	flag.StringVar(targetFlag, "t", "", "Target directory (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goDirDiff [options] <dir1> <dir2>\n")
		fmt.Fprintf(os.Stderr, "       goDirDiff -a [-d rl|lr] [-p <patch_file>] -t <target_dir>\n\n")
		fmt.Fprintf(os.Stderr, "Recursively compares two directories and outputs changes in git unified diff format,\n")
		fmt.Fprintf(os.Stderr, "or applies an exported diff to a target directory:\n")
		fmt.Fprintf(os.Stderr, "  -d rl: bring in right (bottom) changes that belong to dir2 into target directory (default)\n")
		fmt.Fprintf(os.Stderr, "  -d lr: bring in left (top) changes that belong to dir1 into target directory\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	args := flag.Args()

	dirMode, err := diff.ParseDirection(*directionFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	isApplyMode := *applyFlag || *patchPathFlag != "" || *targetFlag != ""

	if isApplyMode {
		handleApply(args, *patchPathFlag, *targetFlag, dirMode)
		return
	}

	// Normal diff comparison mode
	if len(args) != 2 {
		flag.Usage()
		os.Exit(2)
	}

	dirA := args[0]
	dirB := args[1]

	opts := diff.Options{
		ContextLines: *contextFlag,
		Color:        *colorFlag,
		Filter:       *filterFlag,
	}

	diffOutput, hasDiff, err := diff.CompareDirectories(dirA, dirB, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	shouldExport := *exportFlag || *exportPathFlag != ""
	if shouldExport {
		res, err := diff.ExportDiff(diffOutput, *exportPathFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Export error: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Diff exported to %s (SHA-256: %s)\n", res.DiffPath, res.SHA256)
	}

	if hasDiff {
		fmt.Print(diffOutput)
		os.Exit(1)
	}

	os.Exit(0)
}

func handleApply(args []string, patchFlag, targetFlag string, dir diff.PatchDirection) {
	targetDir := targetFlag
	patchFile := patchFlag

	if len(args) == 1 {
		if targetDir == "" {
			targetDir = args[0]
		} else if patchFile == "" {
			patchFile = args[0]
		}
	} else if len(args) >= 2 {
		if patchFile == "" {
			patchFile = args[0]
		}
		if targetDir == "" {
			targetDir = args[1]
		}
	}

	if patchFile == "" {
		patchFile = filepath.Join("exported_diff", "diff.patch")
	}

	if targetDir == "" {
		fmt.Fprintf(os.Stderr, "Error: target directory required for applying patch (use -t <target_dir> or provide target directory)\n")
		os.Exit(2)
	}

	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: target directory %q does not exist or is not a directory\n", targetDir)
		os.Exit(2)
	}

	res, err := diff.ApplyDiffFile(patchFile, targetDir, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Apply error: %v\n", err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "Successfully applied patch %s to %s (direction: %s)\n", patchFile, targetDir, dir)
	printApplySummary(res)
	os.Exit(0)
}

func printApplySummary(res *diff.ApplyResult) {
	if len(res.Modified) > 0 {
		fmt.Fprintf(os.Stderr, "  Modified: %d file(s)\n", len(res.Modified))
	}
	if len(res.Added) > 0 {
		fmt.Fprintf(os.Stderr, "  Added:    %d file(s)\n", len(res.Added))
	}
	if len(res.Deleted) > 0 {
		fmt.Fprintf(os.Stderr, "  Deleted:  %d file(s)\n", len(res.Deleted))
	}
	if len(res.Skipped) > 0 {
		fmt.Fprintf(os.Stderr, "  Skipped:  %d file(s)\n", len(res.Skipped))
	}
}
