// Package gitrepo answers the questions Maslow asks of a git repository:
// which files exist, which changed, and what a file looked like at a commit.
// Every function takes a directory inside the repository; "" means the
// current one. File lists always hold repository-root-relative paths.
package gitrepo

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// HeadSHA returns the commit HEAD points at.
func HeadSHA(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// MergeBase resolves ref and returns its merge base with HEAD.
func MergeBase(dir, ref string) (string, error) {
	out, err := run(dir, "merge-base", ref, "HEAD")
	if err != nil {
		return "", fmt.Errorf("cannot find a merge base between %q and HEAD: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CandidateFiles lists tracked files plus untracked files that are not ignored.
func CandidateFiles(dir string) ([]string, error) {
	out, err := runAtRoot(dir, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return sortedUnique(splitNul(out)), nil
}

// WorkingTreeChanges lists files with staged, unstaged or untracked changes
// relative to HEAD, excluding ignored files.
func WorkingTreeChanges(dir string) ([]string, error) {
	out, err := runAtRoot(dir, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames")
	if err != nil {
		return nil, err
	}

	var changed []string
	for _, entry := range splitNul(out) {
		if len(entry) > 3 {
			changed = append(changed, entry[3:])
		}
	}
	return sortedUnique(changed), nil
}

// ChangedSince lists files that differ between base and the working tree:
// committed changes since base plus anything uncommitted or untracked.
func ChangedSince(dir, base string) ([]string, error) {
	out, err := runAtRoot(dir, "diff", "--name-only", "-z", "--no-renames", base, "HEAD")
	if err != nil {
		return nil, err
	}
	committed := splitNul(out)

	uncommitted, err := WorkingTreeChanges(dir)
	if err != nil {
		return nil, err
	}
	return sortedUnique(append(committed, uncommitted...)), nil
}

// Show returns the content of path at commit.
func Show(dir, commit, path string) ([]byte, error) {
	return run(dir, "show", commit+":"+path)
}

// ExistsAt reports whether path, relative to dir, is a file at commit. It
// separates a missing file from a git failure, which Show cannot.
func ExistsAt(dir, commit, path string) (bool, error) {
	out, err := run(dir, "ls-tree", "--name-only", commit, "--", path)
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// Prefix returns dir's path below the repository root, with a trailing
// slash, or "" at the root.
func Prefix(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-prefix")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runAtRoot runs git from the repository root, so listings cover the whole
// repository and use root-relative paths wherever dir is.
func runAtRoot(dir string, args ...string) ([]byte, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	return run(strings.TrimSpace(string(out)), args...)
}

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func splitNul(out []byte) []string {
	var parts []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func sortedUnique(items []string) []string {
	sort.Strings(items)
	var out []string
	for i, s := range items {
		if i == 0 || s != items[i-1] {
			out = append(out, s)
		}
	}
	return out
}
