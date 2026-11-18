package gitrepo

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Line is one line of a file, numbered from 1 in the version it belongs to:
// the base for a removed line, the working tree for an added one.
type Line struct {
	Number int
	Text   string
}

// Hunk is a run of removed lines and the lines that replaced them.
type Hunk struct {
	Removed []Line
	Added   []Line
}

// FileDiff is one file's line changes. Path is repository-root-relative.
type FileDiff struct {
	Path    string
	Deleted bool
	Hunks   []Hunk
}

// DiffLines returns line changes between base and the working tree, with
// untracked files as wholly added. An empty base means HEAD, so only
// uncommitted work counts, matching WorkingTreeChanges.
func DiffLines(dir, base string) ([]FileDiff, error) {
	if base == "" {
		base = "HEAD"
	}
	out, err := runAtRoot(dir, "-c", "core.quotePath=false", "diff", "--no-color", "--no-ext-diff", "--no-renames", "-U0", base)
	if err != nil {
		return nil, err
	}
	diffs, err := parseUnifiedDiff(out)
	if err != nil {
		return nil, err
	}

	root, err := toplevel(dir)
	if err != nil {
		return nil, err
	}
	untracked, err := run(root, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, path := range sortedUnique(splitNul(untracked)) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, wholeFileAdded(path, string(content)))
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Path < diffs[j].Path })
	return diffs, nil
}

func wholeFileAdded(path, content string) FileDiff {
	if content == "" {
		return FileDiff{Path: path}
	}
	var added []Line
	for i, text := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		added = append(added, Line{Number: i + 1, Text: text})
	}
	return FileDiff{Path: path, Hunks: []Hunk{{Added: added}}}
}

// parseUnifiedDiff reads `git diff -U0` output. Hunk line counts decide where
// a hunk ends, because a removed line can itself start with "--- ".
func parseUnifiedDiff(out []byte) ([]FileDiff, error) {
	var diffs []FileDiff
	var oldPath string
	var oldLeft, newLeft, oldLine, newLine int

	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		text := scanner.Text()
		inHunk := oldLeft > 0 || newLeft > 0
		switch {
		case inHunk && strings.HasPrefix(text, "-"):
			hunk := lastHunk(diffs)
			hunk.Removed = append(hunk.Removed, Line{Number: oldLine, Text: text[1:]})
			oldLine++
			oldLeft--
		case inHunk && strings.HasPrefix(text, "+"):
			hunk := lastHunk(diffs)
			hunk.Added = append(hunk.Added, Line{Number: newLine, Text: text[1:]})
			newLine++
			newLeft--
		case strings.HasPrefix(text, `\`):
		case strings.HasPrefix(text, "diff --git "):
			oldPath = ""
		case strings.HasPrefix(text, "--- "):
			oldPath = diffPath(text[4:])
		case strings.HasPrefix(text, "+++ "):
			newPath := diffPath(text[4:])
			if newPath == "" {
				diffs = append(diffs, FileDiff{Path: oldPath, Deleted: true})
			} else {
				diffs = append(diffs, FileDiff{Path: newPath})
			}
		case strings.HasPrefix(text, "@@ "):
			if len(diffs) == 0 {
				return nil, fmt.Errorf("git diff: hunk before any file header: %q", text)
			}
			var err error
			oldLine, oldLeft, newLine, newLeft, err = parseHunkHeader(text)
			if err != nil {
				return nil, err
			}
			last := &diffs[len(diffs)-1]
			last.Hunks = append(last.Hunks, Hunk{})
		}
	}
	return diffs, scanner.Err()
}

func lastHunk(diffs []FileDiff) *Hunk {
	file := &diffs[len(diffs)-1]
	return &file.Hunks[len(file.Hunks)-1]
}

// diffPath turns a "---"/"+++" operand into a root-relative path, or "" for
// /dev/null. Git appends a tab to names containing spaces and quotes unusual ones.
func diffPath(operand string) string {
	operand = strings.TrimSuffix(operand, "\t")
	if operand == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(operand, `"`) {
		if unquoted, err := strconv.Unquote(operand); err == nil {
			operand = unquoted
		}
	}
	_, path, _ := strings.Cut(operand, "/")
	return path
}

// parseHunkHeader reads "@@ -a[,b] +c[,d] @@"; an omitted count is 1.
func parseHunkHeader(header string) (oldStart, oldCount, newStart, newCount int, err error) {
	fields := strings.Fields(header)
	if len(fields) < 3 {
		return 0, 0, 0, 0, fmt.Errorf("git diff: malformed hunk header %q", header)
	}
	oldStart, oldCount, err = parseRange(strings.TrimPrefix(fields[1], "-"))
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("git diff: malformed hunk header %q: %w", header, err)
	}
	newStart, newCount, err = parseRange(strings.TrimPrefix(fields[2], "+"))
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("git diff: malformed hunk header %q: %w", header, err)
	}
	return oldStart, oldCount, newStart, newCount, nil
}

func parseRange(r string) (start, count int, err error) {
	startText, countText, hasCount := strings.Cut(r, ",")
	if start, err = strconv.Atoi(startText); err != nil {
		return 0, 0, err
	}
	if !hasCount {
		return start, 1, nil
	}
	count, err = strconv.Atoi(countText)
	return start, count, err
}
