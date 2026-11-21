//go:build !unix

package runner

import "os/exec"

// Windows has no process groups to kill, so it keeps the behaviour from before ADR 0044.
func combinedOutputInGroup(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}
