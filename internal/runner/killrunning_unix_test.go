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
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := writeScript(t, "sleep 30 >/dev/null 2>&1 &\necho $! > "+pidFile+"\nwait\n")

	done := make(chan time.Duration, 1)
	start := time.Now()
	go func() {
		runOneCheck(t, script, "30s")
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

func TestKillRunning_NothingRunningIsANoOp(t *testing.T) {
	KillRunning()
	status, _, _ := runOneCheck(t, writeScript(t, "exit 0\n"), "5s")
	if status != "pass" {
		t.Errorf("expected a normal check to pass after KillRunning, got %s", status)
	}
}
