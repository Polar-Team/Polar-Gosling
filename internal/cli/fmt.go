package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polar-gosling/gosling/internal/parser"
	"github.com/spf13/cobra"
)

var (
	fmtPath   string
	fmtCheck  bool
	fmtDiff   bool
	fmtStdout bool
)

// fmtCmd represents the fmt command
var fmtCmd = &cobra.Command{
	Use:   "fmt [file...]",
	Short: "Format .fly configuration files",
	Long: `Format .fly configuration files to canonical style.

Without flags, rewrites files in-place. Use --check to verify formatting
without modifying files, --diff to show a unified diff, or --stdout to
print the formatted output to stdout (single file only).

Examples:
  gosling fmt
  gosling fmt Eggs/my-app/config.fly
  gosling fmt --check
  gosling fmt --diff
  gosling fmt --stdout Eggs/my-app/config.fly`,
	Args: cobra.ArbitraryArgs,
	RunE: runFmt,
}

func init() {
	rootCmd.AddCommand(fmtCmd)
	fmtCmd.Flags().StringVarP(&fmtPath, "path", "p", "", "Path to Nest repository (default: current directory)")
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "Check formatting without modifying files (exit 1 if unformatted)")
	fmtCmd.Flags().BoolVar(&fmtDiff, "diff", false, "Print unified diff of formatting changes")
	fmtCmd.Flags().BoolVar(&fmtStdout, "stdout", false, "Print formatted output to stdout (requires exactly one file argument)")
}

func runFmt(cmd *cobra.Command, args []string) error {
	// --- Flag validation (before any file I/O) ---
	if fmtStdout {
		if len(args) != 1 {
			return fmt.Errorf("--stdout requires exactly one file argument")
		}
		if fmtCheck || fmtDiff {
			return fmt.Errorf("--stdout is mutually exclusive with --check and --diff")
		}
	}

	// --- File discovery ---
	var files []string
	if len(args) > 0 && !fmtStdout {
		// Explicit file list
		for _, arg := range args {
			abs, err := filepath.Abs(arg)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", arg, err)
			}
			files = append(files, abs)
		}
	} else if fmtStdout {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve path %q: %w", args[0], err)
		}
		files = []string{abs}
	} else {
		nestRoot := fmtPath
		if nestRoot == "" {
			var err error
			nestRoot, err = findNestRoot()
			if err != nil {
				return fmt.Errorf("not in a Nest repository: %w\nRun 'gosling init' to create a new Nest repository", err)
			}
		}
		var err error
		files, err = findFlyFiles(nestRoot)
		if err != nil {
			return fmt.Errorf("failed to find .fly files: %w", err)
		}
		if len(files) == 0 {
			fmt.Println("⚠️  No .fly files found in the repository")
			return nil
		}
	}

	// --- Orchestration loop ---
	p := parser.NewParser()
	f := parser.NewFormatter()

	parseErrors := 0
	checkFailures := 0
	formatted := 0
	unchanged := 0

	for _, filePath := range files {
		original, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", filePath, err)
			parseErrors++
			continue
		}

		config, err := p.ParseFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", filePath, err)
			parseErrors++
			continue
		}

		canonical := f.Format(config)
		// Formatter produces no trailing newline; files conventionally end with one.
		canonicalWithNewline := canonical + "\n"

		originalStr := string(original)
		alreadyFormatted := originalStr == canonicalWithNewline

		switch {
		case fmtStdout:
			fmt.Print(canonicalWithNewline)

		case fmtDiff && !alreadyFormatted:
			relPath, _ := filepath.Rel(fmtPath, filePath)
			if relPath == "" || relPath == "." {
				relPath = filePath
			}
			diff := unifiedDiff(originalStr, canonicalWithNewline, "a/"+relPath, "b/"+relPath)
			fmt.Print(diff)
			if fmtCheck {
				checkFailures++
			} else {
				if writeErr := os.WriteFile(filePath, []byte(canonicalWithNewline), 0644); writeErr != nil {
					fmt.Fprintf(os.Stderr, "error writing %s: %v\n", filePath, writeErr)
					parseErrors++
					continue
				}
				formatted++
			}

		case fmtDiff && alreadyFormatted:
			// nothing to print

		case fmtCheck:
			if !alreadyFormatted {
				relPath, _ := filepath.Rel(fmtPath, filePath)
				if relPath == "" || relPath == "." {
					relPath = filePath
				}
				fmt.Fprintf(os.Stderr, "would reformat: %s\n", relPath)
				checkFailures++
			}

		default:
			// In-place mode
			if !alreadyFormatted {
				if writeErr := os.WriteFile(filePath, []byte(canonicalWithNewline), 0644); writeErr != nil {
					fmt.Fprintf(os.Stderr, "error writing %s: %v\n", filePath, writeErr)
					parseErrors++
					continue
				}
				formatted++
			} else {
				unchanged++
			}
		}

		if alreadyFormatted && !fmtStdout && !fmtCheck && !fmtDiff {
			_ = unchanged // already counted above
		}
	}

	// --- Summary (skip for --stdout) ---
	if !fmtStdout {
		printFmtSummary(fmtCheck, fmtDiff, formatted, unchanged, checkFailures, parseErrors, len(files))
	}

	if checkFailures > 0 || parseErrors > 0 {
		return fmt.Errorf("fmt failed")
	}
	return nil
}

func printFmtSummary(check, diff bool, formatted, unchanged, checkFailures, parseErrors, total int) {
	fmt.Println(strings.Repeat("─", 50))
	switch {
	case check:
		fmt.Printf("Summary: %d/%d files need formatting", checkFailures, total)
		if parseErrors > 0 {
			fmt.Printf(", %d parse error(s)", parseErrors)
		}
		fmt.Println()
		if checkFailures == 0 && parseErrors == 0 {
			fmt.Println("✅ All files are properly formatted!")
		}
	case diff:
		if formatted > 0 {
			fmt.Printf("Summary: %d file(s) reformatted", formatted)
		} else {
			fmt.Printf("Summary: %d/%d files need formatting (diff shown above)", checkFailures, total)
		}
		if parseErrors > 0 {
			fmt.Printf(", %d parse error(s)", parseErrors)
		}
		fmt.Println()
	default:
		fmt.Printf("Summary: %d reformatted, %d unchanged", formatted, unchanged)
		if parseErrors > 0 {
			fmt.Printf(", %d parse error(s)", parseErrors)
		}
		fmt.Println()
		if formatted == 0 && parseErrors == 0 {
			fmt.Println("✅ All files are properly formatted!")
		}
	}
}

// unifiedDiff produces a minimal unified diff between oldText and newText.
// oldLabel / newLabel are used for the --- / +++ header lines.
func unifiedDiff(oldText, newText, oldLabel, newLabel string) string {
	if oldText == newText {
		return ""
	}

	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	lcs := computeLCS(oldLines, newLines)
	hunks := buildHunks(oldLines, newLines, lcs)

	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n", oldLabel)
	fmt.Fprintf(&sb, "+++ %s\n", newLabel)
	for _, h := range hunks {
		sb.WriteString(h)
	}
	return sb.String()
}

// splitLines splits text into lines, preserving the newline character at the
// end of each line. A trailing empty string is omitted.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	// SplitAfter on "a\nb\n" yields ["a\n","b\n",""] — drop trailing empty.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// lcsEntry is used during LCS back-tracking.
type lcsEntry struct{ oi, ni int }

// computeLCS returns the longest common subsequence of old/new as a list of
// (oldIndex, newIndex) pairs in ascending order.
func computeLCS(old, new []string) []lcsEntry {
	m, n := len(old), len(new)
	if m == 0 || n == 0 {
		return nil
	}

	// dp[i][j] = length of LCS of old[:i] and new[:j]
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if old[i-1] == new[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Back-track to collect matching pairs
	result := make([]lcsEntry, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if old[i-1] == new[j-1] {
			result = append(result, lcsEntry{i - 1, j - 1})
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}
	// Reverse to get ascending order
	for l, r := 0, len(result)-1; l < r; l, r = l+1, r-1 {
		result[l], result[r] = result[r], result[l]
	}
	return result
}

// hunk groups a contiguous region of changes for unified diff output.
type hunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []string // each prefixed with ' ', '-', or '+'
}

const contextLines = 3

// buildHunks converts LCS matches into unified-diff hunks with context.
func buildHunks(old, new []string, lcs []lcsEntry) []string {
	type edit struct {
		kind    byte // ' ' context, '-' removed, '+' added
		oldLine int  // 0-based index in old (-1 if added)
		newLine int  // 0-based index in new (-1 if removed)
		text    string
	}

	// Build flat edit list
	edits := make([]edit, 0, len(old)+len(new))
	oi, ni, li := 0, 0, 0
	for li <= len(lcs) {
		var matchOI, matchNI int
		if li < len(lcs) {
			matchOI = lcs[li].oi
			matchNI = lcs[li].ni
		} else {
			matchOI = len(old)
			matchNI = len(new)
		}
		// Removals before this match
		for oi < matchOI {
			edits = append(edits, edit{'-', oi, -1, old[oi]})
			oi++
		}
		// Additions before this match
		for ni < matchNI {
			edits = append(edits, edit{'+', -1, ni, new[ni]})
			ni++
		}
		// The matching line itself
		if li < len(lcs) {
			edits = append(edits, edit{' ', oi, ni, old[oi]})
			oi++
			ni++
		}
		li++
	}

	// Group edits into hunks with context
	var hunks []hunk
	i := 0
	for i < len(edits) {
		if edits[i].kind == ' ' {
			i++
			continue
		}
		// Found a change — determine hunk boundaries
		start := i - contextLines
		if start < 0 {
			start = 0
		}
		// Extend end to include all changes + trailing context
		end := i
		for end < len(edits) {
			if edits[end].kind != ' ' {
				end = end + contextLines + 1
				if end > len(edits) {
					end = len(edits)
				}
			} else {
				end++
			}
		}
		// Trim trailing context lines that are beyond the last change
		lastChange := start
		for k := start; k < end; k++ {
			if edits[k].kind != ' ' {
				lastChange = k
			}
		}
		end = lastChange + contextLines + 1
		if end > len(edits) {
			end = len(edits)
		}

		h := hunk{}
		oldStart, newStart := -1, -1
		for k := start; k < end; k++ {
			e := edits[k]
			switch e.kind {
			case ' ':
				if oldStart == -1 {
					oldStart = e.oldLine
					newStart = e.newLine
				}
				h.oldCount++
				h.newCount++
				h.lines = append(h.lines, " "+e.text)
			case '-':
				if oldStart == -1 {
					oldStart = e.oldLine
				}
				if newStart == -1 {
					newStart = ni // will be set properly below
				}
				h.oldCount++
				h.lines = append(h.lines, "-"+e.text)
			case '+':
				if newStart == -1 {
					newStart = e.newLine
				}
				if oldStart == -1 {
					oldStart = oi
				}
				h.newCount++
				h.lines = append(h.lines, "+"+e.text)
			}
		}
		if oldStart < 0 {
			oldStart = 0
		}
		if newStart < 0 {
			newStart = 0
		}
		h.oldStart = oldStart + 1 // unified diff uses 1-based line numbers
		h.newStart = newStart + 1
		hunks = append(hunks, h)
		i = end
	}

	// Render hunks to strings
	result := make([]string, 0, len(hunks))
	for _, h := range hunks {
		var sb strings.Builder
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount)
		for _, line := range h.lines {
			sb.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				sb.WriteString("\n")
			}
		}
		result = append(result, sb.String())
	}
	return result
}
