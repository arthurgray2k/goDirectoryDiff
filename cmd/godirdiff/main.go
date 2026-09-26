package main

import (
	"flag"
	"fmt"
	"os"

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

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goDirDiff [options] <dir1> <dir2>\n\n")
		fmt.Fprintf(os.Stderr, "Recursively compares two directories and outputs changes in git unified diff format.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
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
