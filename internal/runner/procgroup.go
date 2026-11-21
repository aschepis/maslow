package runner

import "time"

// outputWaitDelay bounds how long a step waits for its output after the command exits or is
// killed, so a descendant holding the pipe cannot hang the run (ADR 0044).
var outputWaitDelay = 3 * time.Second
