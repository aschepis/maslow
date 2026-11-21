//go:build unix

package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aschepis/maslow/internal/spec"
)

// writeScript writes an sh script into a temp dir, since a check's run line is split on spaces.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// firstPID reads the pid the script printed on its first line and kills it when the test ends.
func firstPID(t *testing.T, output string) int {
	t.Helper()
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("expected a pid on the first line of output, got %q", output)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	return pid
}

// assertGone polls because an orphan stays visible until init reaps it.
func assertGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("process %d is still running after the step ended", pid)
}

func shortWaitDelay(t *testing.T) {
	t.Helper()
	old := outputWaitDelay
	outputWaitDelay = 200 * time.Millisecond
	t.Cleanup(func() { outputWaitDelay = old })
}

func shortCLITimeout(t *testing.T) {
	t.Helper()
	old := cliStepTimeout
	cliStepTimeout = time.Second
	t.Cleanup(func() { cliStepTimeout = old })
}

func runOneCheck(t *testing.T, run, timeout string) (status, output string, elapsed time.Duration) {
	t.Helper()
	start := time.Now()
	result := runCheck(spec.CheckRunner{Name: "c", Kind: "command", Run: run, Timeout: timeout})
	return result.Status, result.Output, time.Since(start)
}

const (
	detachedChild = "sleep 30 >/dev/null 2>&1 &\necho $!\n"
	// The child inherits stdout and the script waits on it, so only killing the group closes the pipe.
	childHoldsOutputWhileRunning = "sleep 30 &\necho $!\nwait\n"
	// The script exits at once, but the child keeps stdout open.
	childHoldsOutputAfterExit = "sleep 30 &\necho $!\n"
)

func TestRunCheck_BackgroundChildIsKilledWhenCheckExits(t *testing.T) {
	status, output, _ := runOneCheck(t, "sh "+writeScript(t, detachedChild), "10s")

	if status != "pass" {
		t.Fatalf("expected pass, got %s: %s", status, output)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCheck_ChildHoldingOutputEndsAtTimeout(t *testing.T) {
	status, output, elapsed := runOneCheck(t, "sh "+writeScript(t, childHoldsOutputWhileRunning), "1s")

	if status != "fail" {
		t.Errorf("expected fail, got %s", status)
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected the check to end near its 1s timeout, took %s", elapsed)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCheck_ChildHoldingOutputAfterExitDoesNotHang(t *testing.T) {
	shortWaitDelay(t)
	status, output, elapsed := runOneCheck(t, "sh "+writeScript(t, childHoldsOutputAfterExit), "10s")

	if status != "fail" {
		t.Errorf("expected fail, got %s", status)
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected the check to end soon after the script exited, took %s", elapsed)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCheck_NormalCheckIsUnaffected(t *testing.T) {
	status, output, _ := runOneCheck(t, "sh "+writeScript(t, "echo hello\n"), "10s")

	if status != "pass" || strings.TrimSpace(output) != "hello" {
		t.Errorf("expected pass with output hello, got %s %q", status, output)
	}
}

func cliStep(script string) spec.Step {
	return spec.Step{Action: "cli", Command: "sh", Args: []string{script}}
}

func TestRunCLIStep_BackgroundChildIsKilledWhenStepExits(t *testing.T) {
	exitCode, output, err := runCLIStep(cliStep(writeScript(t, detachedChild)), t.TempDir())

	if err != nil || exitCode != 0 {
		t.Fatalf("expected exit 0, got %d, %v", exitCode, err)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCLIStep_ChildHoldingOutputEndsAtTimeout(t *testing.T) {
	shortCLITimeout(t)
	start := time.Now()
	exitCode, output, _ := runCLIStep(cliStep(writeScript(t, childHoldsOutputWhileRunning)), t.TempDir())

	if exitCode == 0 {
		t.Errorf("expected a nonzero exit code after the timeout")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("expected the step to end near its 1s timeout, took %s", elapsed)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCLIStep_ChildHoldingOutputAfterExitDoesNotHang(t *testing.T) {
	shortWaitDelay(t)
	start := time.Now()
	_, output, err := runCLIStep(cliStep(writeScript(t, childHoldsOutputAfterExit)), t.TempDir())

	if err == nil {
		t.Errorf("expected an error for output held open after exit")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("expected the step to end soon after the script exited, took %s", elapsed)
	}
	assertGone(t, firstPID(t, output))
}

func TestRunCLIStep_NormalStepIsUnaffected(t *testing.T) {
	step := spec.Step{Action: "cli", Command: "sh", Args: []string{"-c", "cat; exit 3"}, Stdin: "hi"}
	exitCode, output, err := runCLIStep(step, t.TempDir())

	if err != nil || exitCode != 3 || output != "hi" {
		t.Errorf("expected exit 3 with output hi, got %d %q %v", exitCode, output, err)
	}
}

func TestContractSetup_CannotLeaveABackgroundProcessRunning(t *testing.T) {
	dir := t.TempDir()
	setup := spec.Step{Action: "cli", Command: "sh", Args: []string{"-c", "sleep 30 >/dev/null 2>&1 & echo $! > pid"}}
	contract := spec.Contract{Name: "c", Setup: []spec.Step{setup}, Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps("true")}}}
	runIn(t, dir, contract)

	data, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		t.Fatal(err)
	}
	assertGone(t, firstPID(t, string(data)))
}
