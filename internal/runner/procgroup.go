package runner

import (
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// outputWaitDelay bounds how long a step waits for its output after the command exits or is
// killed, so a descendant holding the pipe cannot hang the run (ADR 0044).
var outputWaitDelay = 3 * time.Second

// combinedOutput runs cmd like cmd.CombinedOutput, and on Unix kills every process it left behind.
func combinedOutput(cmd *exec.Cmd) ([]byte, error) {
	out, err := combinedOutputInGroup(cmd)
	if errors.Is(err, exec.ErrWaitDelay) {
		err = fmt.Errorf("output was still open %s after the command exited; the background processes holding it were killed", outputWaitDelay)
	}
	return out, err
}
