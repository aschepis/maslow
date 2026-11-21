//go:build unix

package runner

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// waitForPID reads the pid a script wrote to path, waiting for the script to get that far.
func waitForPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
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
	t.Fatalf("script never wrote %s", path)
	return 0
}

func TestKillRunning_KillsTheGroupOfARunningCheck(t *testing.T) {
	resetRunning(t)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := writeScript(t, "sleep 30 >/dev/null 2>&1 &\necho $! > "+pidFile+"\nwait\n")

	done := make(chan time.Duration, 1)
	start := time.Now()
	go func() {
		runOneCheck(t, "sh "+script, "30s")
		done <- time.Since(start)
	}()

	pid := waitForPID(t, pidFile)
	KillRunning()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("check was still running 5s after KillRunning")
	}
	assertGone(t, pid)
}

func TestKillRunning_KillsAStepThatStartsAfterward(t *testing.T) {
	resetRunning(t)
	KillRunning()
	status, _, elapsed := runOneCheck(t, "sh "+writeScript(t, "sleep 30\n"), "30s")
	if status != "fail" || elapsed > 5*time.Second {
		t.Errorf("expected a step started after KillRunning to be killed at once, got %s after %s", status, elapsed)
	}
}

// resetRunning undoes KillRunning, which in Maslow is followed by exit and never undone.
func resetRunning(t *testing.T) {
	t.Cleanup(func() {
		running.Lock()
		running.stopping = false
		running.Unlock()
	})
}
