//go:build unix

package runner

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// running holds the process groups of steps in flight, so an interrupt can kill them; a step's
// group is outside the terminal's process group and never sees the user's Ctrl-C.
var running = struct {
	sync.Mutex
	groups   map[int]struct{}
	stopping bool
}{groups: map[int]struct{}{}}

func combinedOutputInGroup(cmd *exec.Cmd) ([]byte, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd.Process.Pid) }
	cmd.WaitDelay = outputWaitDelay

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	pgid := cmd.Process.Pid
	track(pgid)

	err := cmd.Wait()
	_ = killGroup(pgid)
	untrack(pgid)
	return out.Bytes(), err
}

// track kills a group at once if an interrupt already arrived, so no step outlives it.
func track(pgid int) {
	running.Lock()
	defer running.Unlock()
	if running.stopping {
		_ = killGroup(pgid)
		return
	}
	running.groups[pgid] = struct{}{}
}

func untrack(pgid int) {
	running.Lock()
	defer running.Unlock()
	delete(running.groups, pgid)
}

// KillRunning kills the process group of every step still running, and of any that starts
// afterwards, for use when Maslow is interrupted.
func KillRunning() {
	running.Lock()
	defer running.Unlock()
	running.stopping = true
	for pgid := range running.groups {
		_ = killGroup(pgid)
	}
}

// killGroup is safe after the leader is reaped: a pid is not reused while its group has members.
func killGroup(pgid int) error {
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
