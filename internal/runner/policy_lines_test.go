package runner

import (
	"slices"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func TestShrinkOnly_AddedLineFails(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"authz/baseline.txt": "alice\nbob\n"},
		map[string]string{"authz/baseline.txt": "alice\nbob\nmallory\n"})

	results := RunPolicy(&spec.Policy{ShrinkOnly: []string{"authz/*.txt"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %+v", results)
	}
	got := results[0]
	if got.Kind != "shrink_only" || got.Status != "fail" {
		t.Fatalf("expected shrink_only fail, got %+v", got)
	}
	if !slices.Equal(got.Files, []string{"authz/baseline.txt"}) {
		t.Errorf("expected files [authz/baseline.txt], got %v", got.Files)
	}
	if !slices.Equal(got.Lines, []string{"authz/baseline.txt:3: mallory"}) {
		t.Errorf("expected the added line reported, got %v", got.Lines)
	}
}

func TestShrinkOnly_RemovedLinePasses(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"authz/baseline.txt": "alice\nbob\n"},
		map[string]string{"authz/baseline.txt": "alice\n"})

	results := RunPolicy(&spec.Policy{ShrinkOnly: []string{"authz/baseline.txt"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 || results[0].Status != "pass" {
		t.Fatalf("expected one pass, got %+v", results)
	}
}

func TestShrinkOnly_NewUntrackedFileFails(t *testing.T) {
	dir, base := branchWithChange(t, map[string]string{"app.go": "v1"}, map[string]string{"app.go": "v2"})
	writeFile(t, dir, "authz/extra.txt", "eve\n")

	results := RunPolicy(&spec.Policy{ShrinkOnly: []string{"authz/**"}}, PolicyScope{Dir: dir, BaseSHA: base})
	if results[0].Status != "fail" || !slices.Equal(results[0].Lines, []string{"authz/extra.txt:1: eve"}) {
		t.Fatalf("expected a new file to count as added lines, got %+v", results[0])
	}
}

func TestShrinkOnly_ObservedReportsWouldFail(t *testing.T) {
	dir, base := branchWithChange(t,
		map[string]string{"baseline.txt": "a\n"},
		map[string]string{"baseline.txt": "a\nb\n"})

	policy := &spec.Policy{Observe: &spec.Policy{ShrinkOnly: []string{"baseline.txt"}}}
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: base})
	if len(results) != 1 || results[0].Status != "would_fail" || results[0].Mode != "observe" {
		t.Fatalf("expected one observed would_fail, got %+v", results)
	}
}

func testIntegrity(t *testing.T, policy *spec.Policy, base, changed map[string]string) map[string]evidenceView {
	t.Helper()
	dir, baseSHA := branchWithChange(t, base, changed)
	results := RunPolicy(policy, PolicyScope{Dir: dir, BaseSHA: baseSHA})
	byRule := make(map[string]evidenceView, len(results))
	for _, r := range results {
		if r.Kind != "test_integrity" {
			t.Fatalf("unexpected result kind %+v", r)
		}
		byRule[r.Rule] = evidenceView{Status: r.Status, Files: r.Files, Lines: r.Lines, Error: r.Error}
	}
	return byRule
}

type evidenceView struct {
	Status string
	Files  []string
	Lines  []string
	Error  string
}

const goTest = `package a

import "testing"

func TestAlpha(t *testing.T) {}

func TestBeta(t *testing.T) {}
`

func TestTestIntegrity_NewSkipIsFlagged(t *testing.T) {
	skipped := strings.Replace(goTest, "func TestBeta(t *testing.T) {}", "func TestBeta(t *testing.T) {\n\tt.Skip(\"flaky\")\n}", 1)
	got := testIntegrity(t, &spec.Policy{TestIntegrity: &spec.TestIntegrity{}},
		map[string]string{"a/a_test.go": goTest},
		map[string]string{"a/a_test.go": skipped})

	skips := got["added_skips"]
	if skips.Status != "needs_review" {
		t.Fatalf("expected added_skips needs_review, got %+v", got)
	}
	if !slices.Equal(skips.Lines, []string{`a/a_test.go:8: t.Skip("flaky")`}) {
		t.Errorf("expected the skip line reported, got %v", skips.Lines)
	}
	if got["removed_tests"].Status != "pass" {
		t.Errorf("changing TestBeta's body is not removing it, got %+v", got["removed_tests"])
	}
	if got["deleted_test_files"].Status != "pass" {
		t.Errorf("expected deleted_test_files pass, got %+v", got["deleted_test_files"])
	}
}

func TestTestIntegrity_RemovedTestAndDeletedFile(t *testing.T) {
	withoutBeta := strings.Replace(goTest, "func TestBeta(t *testing.T) {}\n", "", 1)
	dir, base := branchWithChange(t,
		map[string]string{"a/a_test.go": goTest, "b/b_test.py": "def test_one():\n    pass\n", "keep.go": "package a\n"},
		map[string]string{"a/a_test.go": withoutBeta})
	runGit(t, dir, "rm", "-q", "b/b_test.py")

	results := RunPolicy(&spec.Policy{TestIntegrity: &spec.TestIntegrity{}}, PolicyScope{Dir: dir, BaseSHA: base})
	got := map[string][]string{}
	status := map[string]string{}
	for _, r := range results {
		got[r.Rule] = append(r.Files, r.Lines...)
		status[r.Rule] = r.Status
	}
	if status["removed_tests"] != "needs_review" || !slices.Contains(got["removed_tests"], "a/a_test.go:7: func TestBeta(t *testing.T) {}") {
		t.Errorf("expected TestBeta reported as removed, got %s %v", status["removed_tests"], got["removed_tests"])
	}
	if !slices.Contains(got["removed_tests"], "b/b_test.py:1: def test_one():") {
		t.Errorf("expected test_one from the deleted python file reported, got %v", got["removed_tests"])
	}
	if status["deleted_test_files"] != "needs_review" || !slices.Contains(got["deleted_test_files"], "b/b_test.py") {
		t.Errorf("expected b/b_test.py reported as deleted, got %s %v", status["deleted_test_files"], got["deleted_test_files"])
	}
}

func TestTestIntegrity_MovedTestIsNotRemoved(t *testing.T) {
	withoutBeta := strings.Replace(goTest, "func TestBeta(t *testing.T) {}\n", "", 1)
	got := testIntegrity(t, &spec.Policy{TestIntegrity: &spec.TestIntegrity{}},
		map[string]string{"a/a_test.go": goTest},
		map[string]string{"a/a_test.go": withoutBeta, "a/b_test.go": "package a\n\nimport \"testing\"\n\nfunc TestBeta(t *testing.T) {\n}\n"})
	if got["removed_tests"].Status != "pass" {
		t.Errorf("a test moved to another file is not removed, got %+v", got["removed_tests"])
	}
}

func TestTestIntegrity_SkipOutsideTestFilesIgnored(t *testing.T) {
	got := testIntegrity(t, &spec.Policy{TestIntegrity: &spec.TestIntegrity{}},
		map[string]string{"helper.go": "package a\n"},
		map[string]string{"helper.go": "package a\n\nfunc skip(t T) { t.Skip() }\n"})
	if got["added_skips"].Status != "pass" {
		t.Errorf("only test files count, got %+v", got["added_skips"])
	}
}

func TestTestIntegrity_DefaultLanguageSkips(t *testing.T) {
	cases := map[string][2]string{
		"spec/a_spec.rb":  {"it 'works' do\nend\n", "xit 'works' do\nend\n"},
		"web/a.test.ts":   {"it('works', () => {})\n", "it.skip('works', () => {})\n"},
		"tests/test_a.py": {"def test_a():\n    pass\n", "@pytest.mark.skip\ndef test_a():\n    pass\n"},
	}
	for path, c := range cases {
		got := testIntegrity(t, &spec.Policy{TestIntegrity: &spec.TestIntegrity{}},
			map[string]string{path: c[0]}, map[string]string{path: c[1]})
		if got["added_skips"].Status != "needs_review" {
			t.Errorf("%s: expected the new skip flagged, got %+v", path, got["added_skips"])
		}
	}
}

func TestTestIntegrity_ConfiguredLanguageReplacesDefaults(t *testing.T) {
	policy := &spec.Policy{TestIntegrity: &spec.TestIntegrity{Languages: map[string]spec.TestLanguage{
		"elixir": {Files: []string{"test/**/*_test.exs"}, Tests: []string{`^\s*test\s+"[^"]*"`}, Skips: []string{`@tag :skip`}},
	}}}
	got := testIntegrity(t, policy,
		map[string]string{"test/a_test.exs": "test \"a\" do\nend\n", "a/a_test.go": goTest},
		map[string]string{"test/a_test.exs": "@tag :skip\ntest \"a\" do\nend\n", "a/a_test.go": "package a\n"})
	if got["added_skips"].Status != "needs_review" {
		t.Errorf("expected the elixir skip flagged, got %+v", got["added_skips"])
	}
	if got["removed_tests"].Status != "pass" {
		t.Errorf("go is not configured, so its removed tests are not judged, got %+v", got["removed_tests"])
	}
}

func TestTestIntegrity_BuiltinFillsUnsetFields(t *testing.T) {
	policy := &spec.Policy{TestIntegrity: &spec.TestIntegrity{Languages: map[string]spec.TestLanguage{
		"go": {Skips: []string{`\bskipUnlessIntegration\(`}},
	}}}
	got := testIntegrity(t, policy,
		map[string]string{"a/a_test.go": goTest},
		map[string]string{"a/a_test.go": strings.Replace(goTest, "{}\n\nfunc TestBeta", "{\n\tskipUnlessIntegration(t)\n}\n\nfunc TestBeta", 1)})
	if got["added_skips"].Status != "needs_review" {
		t.Errorf("expected the configured go skip flagged with default go files, got %+v", got)
	}
}

func TestTestIntegrity_LoweredThreshold(t *testing.T) {
	policy := &spec.Policy{TestIntegrity: &spec.TestIntegrity{Thresholds: []string{"coverage.yml"}}}
	got := testIntegrity(t, policy,
		map[string]string{"coverage.yml": "min_coverage: 80\nmax_flaky: 2\n"},
		map[string]string{"coverage.yml": "min_coverage: 72.5\nmax_flaky: 3\n"})
	lowered := got["lowered_thresholds"]
	if lowered.Status != "needs_review" {
		t.Fatalf("expected lowered_thresholds needs_review, got %+v", got)
	}
	if !slices.Equal(lowered.Lines, []string{"coverage.yml:1: min_coverage: 80 -> min_coverage: 72.5"}) {
		t.Errorf("expected only the lowered line, got %v", lowered.Lines)
	}
}

func TestTestIntegrity_InvalidRegexIsError(t *testing.T) {
	policy := &spec.Policy{TestIntegrity: &spec.TestIntegrity{Languages: map[string]spec.TestLanguage{
		"go": {Skips: []string{"("}},
	}}}
	got := testIntegrity(t, policy, map[string]string{"a": "1"}, map[string]string{"a": "2"})
	r := got["test_integrity"]
	if r.Status != "error" || !strings.Contains(r.Error, "policy.test_integrity.languages.go.skips") {
		t.Fatalf("expected an error naming the spec section, got %+v", got)
	}
}
