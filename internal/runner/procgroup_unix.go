//go:build unix

package runner

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func combinedOutputInGroup(cmd *exec.Cmd) ([]byte, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd.Process.Pid) }
	cmd.WaitDelay = outputWaitDelay

	out, err := cmd.CombinedOutput()
	if cmd.Process != nil {
		_ = killGroup(cmd.Process.Pid)
	}
	return out, err
}

// killGroup is safe after the leader is reaped: a pid is not reused while its group has members.
func killGroup(pgid int) error {
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
