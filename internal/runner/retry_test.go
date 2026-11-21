package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// Each script appends a line to runs, so a test can count how often it ran.
const (
	passScript     = "echo run >> runs; echo ok"
	failScript     = "echo run >> runs; echo nope; exit 1"
	failOnceScript = "echo run >> runs; if [ -f marker ]; then echo second; exit 0; fi; touch marker; echo first; exit 1"
)

// scriptCheck writes body to a script in a fresh directory and returns a
// check that runs it there, and the directory.
func scriptCheck(t *testing.T, body string, retries int) (spec.CheckRunner, string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(script, []byte("cd "+dir+"\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return spec.CheckRunner{Name: "c", Kind: "script", Run: "sh " + script, Retries: retries}, dir
}

func runCount(t *testing.T, dir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "runs"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "run\n")
}

func attemptStatuses[T any](attempts []T, status func(T) string) []string {
	var out []string
	for _, a := range attempts {
		out = append(out, status(a))
	}
	return out
}

func checkStatuses(r evidence.CheckResult) string {
	return strings.Join(attemptStatuses(r.Attempts, func(a evidence.CheckAttempt) string { return a.Status }), ",")
}

func TestRetries_CheckPassingFirstTimeRunsOnce(t *testing.T) {
	check, dir := scriptCheck(t, passScript, 2)
	got := RunChecks([]spec.CheckRunner{check}, false)[0]

	if got.Status != "pass" || got.Attempts != nil {
		t.Errorf("expected pass with no attempts, got %+v", got)
	}
	if n := runCount(t, dir); n != 1 {
		t.Errorf("expected one run, got %d", n)
	}
}

func TestRetries_CheckFailingThenPassingIsFlaky(t *testing.T) {
	check, dir := scriptCheck(t, failOnceScript, 2)
	got := RunChecks([]spec.CheckRunner{check}, false)[0]

	if got.Status != evidence.StatusFlaky {
		t.Errorf("expected flaky, got %q", got.Status)
	}
	if s := checkStatuses(got); s != "fail,pass" {
		t.Fatalf("expected attempts fail,pass, got %q", s)
	}
	if !strings.Contains(got.Attempts[0].Output, "first") || got.Attempts[0].Error == "" {
		t.Errorf("expected the failed attempt's output and error kept, got %+v", got.Attempts[0])
	}
	if got.Error != "" || !strings.Contains(got.Output, "second") {
		t.Errorf("expected top-level fields to describe the passing attempt, got %+v", got)
	}
	if sum := got.Attempts[0].Duration + got.Attempts[1].Duration; got.Duration < sum {
		t.Errorf("expected the total duration %v to cover every attempt (%v)", got.Duration, sum)
	}
	if n := runCount(t, dir); n != 2 {
		t.Errorf("expected two runs, got %d", n)
	}
}

func TestRetries_CheckFailingEveryTimeFails(t *testing.T) {
	check, dir := scriptCheck(t, failScript, 2)
	got := RunChecks([]spec.CheckRunner{check}, false)[0]

	if got.Status != "fail" {
		t.Errorf("expected fail, got %q", got.Status)
	}
	if s := checkStatuses(got); s != "fail,fail,fail" {
		t.Errorf("expected three failed attempts, got %q", s)
	}
	for i, a := range got.Attempts {
		if !strings.Contains(a.Output, "nope") {
			t.Errorf("attempt %d lost its output: %+v", i, a)
		}
	}
	if n := runCount(t, dir); n != 3 {
		t.Errorf("expected three runs, got %d", n)
	}
}

func TestRetries_CheckWithoutRetriesIsUnchanged(t *testing.T) {
	check, dir := scriptCheck(t, failScript, 0)
	got := RunChecks([]spec.CheckRunner{check}, false)[0]

	if got.Status != "fail" || got.Attempts != nil {
		t.Errorf("expected fail with no attempts, got %+v", got)
	}
	if n := runCount(t, dir); n != 1 {
		t.Errorf("expected one run, got %d", n)
	}
}

// An error is a configuration problem a retry cannot fix.
func TestRetries_CheckErrorIsNotRetried(t *testing.T) {
	got := RunChecks([]spec.CheckRunner{{Name: "empty", Kind: "command", Run: "", Retries: 2}}, false)[0]

	if got.Status != "error" || got.Attempts != nil {
		t.Errorf("expected one error with no attempts, got %+v", got)
	}
}

func TestFinalStatus(t *testing.T) {
	cases := []struct {
		attempts []string
		want     string
	}{
		{[]string{"pass"}, "pass"},
		{[]string{"fail"}, "fail"},
		{[]string{"error"}, "error"},
		{[]string{"fail", "pass"}, evidence.StatusFlaky},
		{[]string{"fail", "fail", "pass"}, evidence.StatusFlaky},
		{[]string{"fail", "fail"}, "fail"},
		{[]string{"fail", "error"}, "error"},
	}
	for _, c := range cases {
		if got := finalStatus(c.attempts); got != c.want {
			t.Errorf("finalStatus(%v) = %q, want %q", c.attempts, got, c.want)
		}
	}
}
