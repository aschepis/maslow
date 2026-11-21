package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aschepis/maslow/internal/progress"
)

func TestProgressPrinter_FormatsLines(t *testing.T) {
	var out strings.Builder
	emit := newProgressPrinter(&out)
	emit(progress.Event{Phase: progress.Plan, Total: 4})
	emit(progress.Event{Phase: progress.Start, Kind: "check", Name: "go-test"})
	emit(progress.Event{Phase: progress.Finish, Kind: "check", Name: "go-test", Statuses: []string{"pass"}, Attempts: 1, Duration: 14200 * time.Millisecond})
	emit(progress.Event{Phase: progress.Start, Kind: "contract", Name: "cli-validate"})
	emit(progress.Event{Phase: progress.Finish, Kind: "contract", Name: "cli-validate", Statuses: []string{"flaky"}, Attempts: 2, Duration: 3100 * time.Millisecond})
	emit(progress.Event{Phase: progress.Start, Kind: "contract", Name: "cli"})
	emit(progress.Event{Phase: progress.Finish, Kind: "contract", Name: "cli", Statuses: []string{"pass", "fail", "pass"}, Attempts: 1, Duration: 500 * time.Millisecond})
	emit(progress.Event{Phase: progress.Start, Kind: "policy"})
	emit(progress.Event{Phase: progress.Finish, Kind: "policy", Statuses: []string{"pass", "needs_review"}, Attempts: 1})

	want := `[1/4] check go-test …
[1/4] check go-test pass (14.2s)
[2/4] contract cli-validate …
[2/4] contract cli-validate flaky (2 attempts, 3.1s)
[3/4] contract cli …
[3/4] contract cli 2 pass, 1 fail (0.5s)
[4/4] policy …
[4/4] policy 1 pass, 1 needs_review (0.0s)
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestProgressFor_Modes(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if emit, err := progressFor("never", file); err != nil || emit != nil {
		t.Errorf("never: got emit=%v err=%v, want nil, nil", emit != nil, err)
	}
	if emit, err := progressFor("auto", file); err != nil || emit != nil {
		t.Errorf("auto on a regular file: got emit=%v err=%v, want nil, nil", emit != nil, err)
	}
	if emit, err := progressFor("always", file); err != nil || emit == nil {
		t.Errorf("always: got emit=%v err=%v, want a printer", emit != nil, err)
	}
	if _, err := progressFor("sometimes", file); err == nil {
		t.Error("sometimes: expected an error")
	}
}

func TestProgressNever_PrintsNothing(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	emit, _ := progressFor("never", file)
	emit.Emit(progress.Event{Phase: progress.Plan, Total: 1})
	emit.Emit(progress.Event{Phase: progress.Start, Kind: "check", Name: "x"})
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("never wrote %d bytes", info.Size())
	}
}

func TestInvalidProgressIsUsageError(t *testing.T) {
	if code := cmdVerify([]string{"--progress", "sometimes"}); code != 2 {
		t.Errorf("verify: expected exit 2, got %d", code)
	}
	if code := cmdAudit([]string{"--progress", "sometimes"}); code != 2 {
		t.Errorf("audit: expected exit 2, got %d", code)
	}
}
