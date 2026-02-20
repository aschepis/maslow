package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Detach creates the sentinel file that prevents future harness updates.
// Idempotent: succeeds silently if already detached.
func Detach(opts Options) error {
	dir := opts.dir()
	w := opts.stdout()

	sentinelPath := filepath.Join(dir, SentinelFile)

	if IsDetached(dir) {
		fmt.Fprintf(w, "harness detach: already detached (%s exists)\n", SentinelFile)
		return nil
	}

	if opts.DryRun {
		fmt.Fprintf(w, "harness detach: would create %s\n", SentinelFile)
		return nil
	}

	content := fmt.Sprintf(`# Maslow Harness Detached
#
# This file was created by 'maslow harness detach' on %s.
# While this file exists, 'maslow harness update' will refuse to run
# (unless --force is used).
#
# To re-enable harness updates, delete this file.
`, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(sentinelPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("harness detach: failed to write %s: %w", SentinelFile, err)
	}

	fmt.Fprintf(w, "harness detach: created %s — harness updates are now blocked\n", SentinelFile)
	return nil
}
