//go:build unix

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const slowSpec = `mas: "1.0"
project:
  name: demo
checks:
  runner:
    - name: slow
      kind: command
      run: "sh %SCRIPT%"
      timeout: 60s
profiles:
  full:
    checks: [slow]
`

// TestVerify_InterruptKillsTheRunningCheck runs the real binary, because the signal handling
// lives in main and the check runs in a process group the terminal's signal cannot reach.
func TestVerify_InterruptKillsTheRunningCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "maslow")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(repo, "child.pid")
	script := filepath.Join(repo, "slow.sh")
	if err := os.WriteFile(script, []byte("sleep 30 >/dev/null 2>&1 &\necho $! > "+pidFile+"\nwait\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "maslow.yaml"), []byte(strings.ReplaceAll(slowSpec, "%SCRIPT%", script)), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	reportPath := filepath.Join(dir, "verify.json")
	cmd := exec.Command(bin, "verify", "--profile", "full", "--progress", "never", "--report", reportPath)
	cmd.Dir = repo
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	pid := waitForChildPID(t, pidFile)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatal(err)
	}

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("maslow was still running 5s after SIGINT")
	}

	if code := cmd.ProcessState.ExitCode(); code != 130 {
		t.Errorf("expected exit 130 after SIGINT, got %d; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "maslow: interrupted") {
		t.Errorf("expected an interrupted message on stderr, got:\n%s", stderr.String())
	}
	if _, err := os.Stat(reportPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no evidence file after an interrupt, stat err: %v", err)
	}
	assertProcessGone(t, pid)
}

func waitForChildPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && strings.HasSuffix(string(data), "\n") {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				t.Fatalf("bad pid in %s: %q", path, data)
			}
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("check never wrote %s", path)
	return 0
}

// assertProcessGone polls because an orphan stays visible until init reaps it.
func assertProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("process %d is still running after maslow was interrupted", pid)
}
