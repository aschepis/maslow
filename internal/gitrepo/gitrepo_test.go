package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Every listing must use repository-root paths and cover the whole
// repository, whichever subdirectory Maslow runs in.
func TestListingsAreRootRelativeFromASubdirectory(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "t@t")
	git(t, dir, "config", "user.name", "t")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "top.txt", "v1")
	write(t, dir, "sub/a.txt", "v1")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	base := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "sub/a.txt", "v2")
	write(t, dir, "top.txt", "v2")
	git(t, dir, "commit", "-qam", "change")
	write(t, dir, "sub/new.txt", "untracked")

	sub := filepath.Join(dir, "sub")

	files, err := CandidateFiles(sub)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"top.txt", "sub/a.txt", "sub/new.txt"} {
		if !slices.Contains(files, want) {
			t.Errorf("CandidateFiles from sub: missing %q in %v", want, files)
		}
	}

	changed, err := ChangedSince(sub, base)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sub/a.txt", "sub/new.txt", "top.txt"}
	if !slices.Equal(changed, want) {
		t.Errorf("ChangedSince from sub = %v, want %v", changed, want)
	}

	prefix, err := Prefix(sub)
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "sub/" {
		t.Errorf("Prefix(sub) = %q, want sub/", prefix)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
