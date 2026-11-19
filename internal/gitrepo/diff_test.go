package gitrepo

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "t@t")
	git(t, dir, "config", "user.name", "t")
	git(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func byPath(diffs []FileDiff) map[string]FileDiff {
	m := make(map[string]FileDiff, len(diffs))
	for _, d := range diffs {
		m[d.Path] = d
	}
	return m
}

func TestDiffLines_CommittedUncommittedAndUntracked(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "sub/a.txt", "one\ntwo\nthree\n")
	write(t, dir, "gone.txt", "bye\n")
	write(t, dir, "dashes.txt", "-- comment\nkeep\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	base := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))

	write(t, dir, "sub/a.txt", "one\nTWO\nthree\nfour\n")
	git(t, dir, "rm", "-q", "gone.txt")
	git(t, dir, "commit", "-qam", "change")
	write(t, dir, "dashes.txt", "keep\n")
	write(t, dir, "sub/new.txt", "x\ny")

	diffs, err := DiffLines(filepath.Join(dir, "sub"), base)
	if err != nil {
		t.Fatal(err)
	}
	got := byPath(diffs)

	a := got["sub/a.txt"]
	wantHunks := []Hunk{
		{Removed: []Line{{2, "two"}}, Added: []Line{{2, "TWO"}}},
		{Added: []Line{{4, "four"}}},
	}
	if a.Deleted || !reflect.DeepEqual(a.Hunks, wantHunks) {
		t.Errorf("sub/a.txt = %+v, want hunks %+v", a, wantHunks)
	}

	if gone := got["gone.txt"]; !gone.Deleted || len(gone.Hunks) != 1 || gone.Hunks[0].Removed[0].Text != "bye" {
		t.Errorf("gone.txt = %+v, want a deletion removing \"bye\"", gone)
	}

	dashes := got["dashes.txt"]
	if len(dashes.Hunks) != 1 || !reflect.DeepEqual(dashes.Hunks[0].Removed, []Line{{1, "-- comment"}}) {
		t.Errorf("a removed line starting with \"-- \" must stay a line, got %+v", dashes)
	}

	newFile := got["sub/new.txt"]
	if len(newFile.Hunks) != 1 || !reflect.DeepEqual(newFile.Hunks[0].Added, []Line{{1, "x"}, {2, "y"}}) {
		t.Errorf("an untracked file must be wholly added, got %+v", newFile)
	}
}

func TestDiffLines_WithoutBaseOnlyUncommitted(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	write(t, dir, "a.txt", "one\ntwo\n")
	git(t, dir, "commit", "-qam", "committed")
	write(t, dir, "a.txt", "one\ntwo\nthree\n")

	diffs, err := DiffLines(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || !reflect.DeepEqual(diffs[0].Hunks, []Hunk{{Added: []Line{{3, "three"}}}}) {
		t.Errorf("expected only the uncommitted line, got %+v", diffs)
	}
}

func TestDiffLines_PathWithSpace(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "my file.txt", "a\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	write(t, dir, "my file.txt", "a\nb\n")

	diffs, err := DiffLines(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Path != "my file.txt" {
		t.Errorf("expected path \"my file.txt\", got %+v", diffs)
	}
}
