package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/spec"
)

func TestRunPolicy_Nil(t *testing.T) {
	results := RunPolicy(nil, PolicyScope{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil policy, got %d", len(results))
	}
}

func TestRunPolicy_Empty(t *testing.T) {
	results := RunPolicy(&spec.Policy{}, PolicyScope{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty policy, got %d", len(results))
	}
}

func TestCheckDenyRules_NoMatches(t *testing.T) {
	results := checkDenyRules([]string{"vendor/**"}, []string{"main.go", "internal/a.go"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for non-matching deny pattern, got %s", results[0].Status)
	}
}

func TestCheckDenyRules_MatchesAtAnyDepth(t *testing.T) {
	for _, file := range []string{"a/.env", "a/b/.env", "a/b/c/.env"} {
		results := checkDenyRules([]string{"**/.env"}, []string{"main.go", file})
		if results[0].Status != "fail" {
			t.Errorf("expected **/.env to fail on %s, got %s", file, results[0].Status)
		}
	}
	for _, file := range []string{"vendor/a.go", "vendor/a/b.go", "vendor/a/b/c.go"} {
		results := checkDenyRules([]string{"vendor/**"}, []string{file})
		if results[0].Status != "fail" {
			t.Errorf("expected vendor/** to fail on %s, got %s", file, results[0].Status)
		}
	}
}

func TestCheckDenyRules_ReportsMatchedFiles(t *testing.T) {
	results := checkDenyRules([]string{"**/secret*"}, []string{"secret.txt", "public.txt", "x/secret.key"})
	if results[0].Status != "fail" {
		t.Fatalf("expected fail, got %s", results[0].Status)
	}
	want := []string{"secret.txt", "x/secret.key"}
	if len(results[0].Files) != len(want) || results[0].Files[0] != want[0] || results[0].Files[1] != want[1] {
		t.Errorf("expected matched files %v, got %v", want, results[0].Files)
	}
}

func TestCheckDenyRules_InvalidPattern(t *testing.T) {
	results := checkDenyRules([]string{"a/["}, []string{"a/b"})
	if results[0].Status != "error" {
		t.Errorf("expected error for malformed pattern, got %s", results[0].Status)
	}
}

func TestCandidateFiles_IgnoresGitignoredFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".gitignore", ".env\n")
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	writeFile(t, dir, "a/b/.env", "SECRET=1")

	files, err := gitrepo.CandidateFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkDenyRules([]string{"**/.env"}, files)
	if results[0].Status != "pass" {
		t.Errorf("expected an ignored, untracked .env to pass, got %s: %s", results[0].Status, results[0].Detail)
	}
}

func TestCandidateFiles_IncludesTrackedAndUnignoredFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".gitignore", ".env\n")
	writeFile(t, dir, "a/.env", "SECRET=1")
	runGit(t, dir, "add", ".gitignore")
	runGit(t, dir, "add", "-f", "a/.env")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "b/credentials.json", "{}")

	files, err := gitrepo.CandidateFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkDenyRules([]string{"**/.env", "**/credentials*"}, files)
	if results[0].Status != "fail" {
		t.Errorf("expected a tracked .env to fail even though it is ignored, got %s", results[0].Status)
	}
	if results[1].Status != "fail" {
		t.Errorf("expected an untracked, unignored credentials file to fail, got %s", results[1].Status)
	}
}

func TestProtected_Unchanged(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	changed, err := gitrepo.WorkingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for unchanged protected file, got %s: %s %s", results[0].Status, results[0].Detail, results[0].Error)
	}
}

func TestProtected_Modified(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "protected.txt", "modified")

	changed, err := gitrepo.WorkingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected fail for modified protected file, got %s", results[0].Status)
	}
}

func TestProtected_GlobMatchesNestedChange(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "docs/a/b/spec.md", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "docs/a/b/spec.md", "modified")

	changed, err := gitrepo.WorkingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"docs/**"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected docs/** to catch a nested change, got %s", results[0].Status)
	}
	if len(results[0].Files) != 1 || results[0].Files[0] != "docs/a/b/spec.md" {
		t.Errorf("expected matched file docs/a/b/spec.md, got %v", results[0].Files)
	}
}

func TestProtected_UntrackedNewFileFails(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "protected.txt", "new")

	changed, err := gitrepo.WorkingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected fail for a new untracked protected file, got %s", results[0].Status)
	}
}

func TestRunPolicy_CommittedProtectedChange(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	base := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))

	runGit(t, dir, "checkout", "-q", "-b", "feature")
	writeFile(t, dir, "protected.txt", "weakened")
	runGit(t, dir, "commit", "-qam", "edit protected file")

	policy := &spec.Policy{Protected: []string{"protected.txt"}}

	withoutBase := RunPolicy(policy, PolicyScope{Dir: dir})
	if withoutBase[0].Status != "pass" {
		t.Errorf("without a base only uncommitted changes count; expected pass, got %s", withoutBase[0].Status)
	}

	withBase := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})
	if withBase[0].Status != "fail" {
		t.Fatalf("expected a committed change to a protected file to fail against the base, got %s", withBase[0].Status)
	}
	if len(withBase[0].Files) != 1 || withBase[0].Files[0] != "protected.txt" {
		t.Errorf("expected matched file protected.txt, got %v", withBase[0].Files)
	}
}

func TestRunPolicy_BaseAlsoCountsUncommittedChanges(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	base := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	writeFile(t, dir, "protected.txt", "uncommitted edit")

	results := RunPolicy(&spec.Policy{Protected: []string{"protected.txt"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if results[0].Status != "fail" {
		t.Errorf("expected an uncommitted change to fail with a base too, got %s", results[0].Status)
	}
}

func TestRunPolicy_DeletedProtectedFileFailsAgainstBase(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	base := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	runGit(t, dir, "rm", "-q", "protected.txt")
	runGit(t, dir, "commit", "-qm", "delete protected file")

	results := RunPolicy(&spec.Policy{Protected: []string{"protected.txt"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if results[0].Status != "fail" {
		t.Errorf("expected deleting a protected file to fail, got %s", results[0].Status)
	}
}

// branchWithChange commits base files on main, then commits changed files on a
// feature branch, and returns the repository and the base SHA.
func branchWithChange(t *testing.T, base, changed map[string]string) (string, string) {
	t.Helper()
	dir := newGitRepo(t)
	for name, content := range base {
		writeFile(t, dir, name, content)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-qm", "base")
	baseSHA := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	runGit(t, dir, "checkout", "-qb", "feature")
	for name, content := range changed {
		writeFile(t, dir, name, content)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-qm", "change")
	return dir, baseSHA
}

func TestGated_ChangeNeedsReviewAndNamesOwner(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"infra/prod/main.tf": "v1", "app.go": "v1"},
		map[string]string{"infra/prod/main.tf": "v2"})

	policy := &spec.Policy{Gated: []spec.GatedDir{{Path: "infra/**", Owner: "@acme/platform"}}}
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %+v", results)
	}
	got := results[0]
	if got.Kind != "gated" || got.Status != "needs_review" {
		t.Fatalf("expected gated needs_review, got %+v", got)
	}
	if got.Owner != "@acme/platform" {
		t.Errorf("expected owner @acme/platform, got %q", got.Owner)
	}
	if len(got.Files) != 1 || got.Files[0] != "infra/prod/main.tf" {
		t.Errorf("expected matched file infra/prod/main.tf, got %v", got.Files)
	}
}

func TestGated_UntouchedPathPasses(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"infra/main.tf": "v1", "app.go": "v1"},
		map[string]string{"app.go": "v2"})

	results := RunPolicy(&spec.Policy{Gated: []spec.GatedDir{{Path: "infra/**", Owner: "ops"}}}, PolicyScope{Dir: dir, BaseSHA: base})
	if results[0].Status != "pass" {
		t.Errorf("expected pass when no gated path changed, got %+v", results[0])
	}
}

func TestAllow_ChangeOutsideAllowedPathsFails(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"apps/web/a.ts": "v1", "Makefile": "v1"},
		map[string]string{"apps/web/a.ts": "v2", "Makefile": "v2"})

	results := RunPolicy(&spec.Policy{Allow: []string{"apps/**", "packages/**"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 {
		t.Fatalf("expected 1 allow result, got %+v", results)
	}
	if results[0].Kind != "allow" || results[0].Status != "fail" {
		t.Fatalf("expected allow fail, got %+v", results[0])
	}
	if len(results[0].Files) != 1 || results[0].Files[0] != "Makefile" {
		t.Errorf("expected only Makefile reported as out of scope, got %v", results[0].Files)
	}
}

func TestAllow_ChangesInsideAllowedPathsPass(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"apps/web/a.ts": "v1", "Makefile": "v1"},
		map[string]string{"apps/web/a.ts": "v2", "apps/api/new.go": "new"})

	results := RunPolicy(&spec.Policy{Allow: []string{"apps/**"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %+v", results[0])
	}
}

func TestObserve_ViolationsReportWhatEnforcementWouldDo(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"locked.txt": "v1", "infra/main.tf": "v1", "app.go": "v1"},
		map[string]string{"locked.txt": "v2", "infra/main.tf": "v2"})

	policy := &spec.Policy{Observe: &spec.Policy{
		Protected: []string{"locked.txt"},
		Gated:     []spec.GatedDir{{Path: "infra/**", Owner: "ops"}},
		Allow:     []string{"infra/**"},
		Deny:      []string{"app.go"},
	}}
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})

	want := map[string]string{"deny": "would_fail", "protected": "would_fail", "gated": "would_need_review", "allow": "would_fail"}
	if len(results) != len(want) {
		t.Fatalf("expected %d observed results, got %+v", len(want), results)
	}
	for _, r := range results {
		if r.Mode != "observe" {
			t.Errorf("%s: expected mode observe, got %q", r.Kind, r.Mode)
		}
		if r.Status != want[r.Kind] {
			t.Errorf("%s: expected %s, got %+v", r.Kind, want[r.Kind], r)
		}
		if len(r.Files) == 0 {
			t.Errorf("%s: expected the files it would have flagged, got none", r.Kind)
		}
	}
}

func TestObserve_SameRuleEnforcedFails(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"locked.txt": "v1"},
		map[string]string{"locked.txt": "v2"})

	results := RunPolicy(&spec.Policy{Protected: []string{"locked.txt"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 || results[0].Status != "fail" || results[0].Mode != "" {
		t.Fatalf("expected one enforced fail, got %+v", results)
	}
}

func TestObserve_CleanRulePassesInObserveMode(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"locked.txt": "v1", "app.go": "v1"},
		map[string]string{"app.go": "v2"})

	policy := &spec.Policy{Observe: &spec.Policy{Protected: []string{"locked.txt"}}}
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 || results[0].Status != "pass" || results[0].Mode != "observe" {
		t.Fatalf("expected one observed pass, got %+v", results)
	}
}

func TestObserve_InvalidPatternIsObservedError(t *testing.T) {
	dir, base := branchWithChange(t, map[string]string{"a": "1"}, map[string]string{"a": "2"})

	policy := &spec.Policy{Observe: &spec.Policy{Protected: []string{"[bad"}}}
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 || results[0].Status != "would_error" || results[0].Error == "" {
		t.Fatalf("expected one would_error with a message, got %+v", results)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(out)
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
