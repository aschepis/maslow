package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// Update updates harness files to the latest version.
// Files that don't exist are created. Files that match the latest content are skipped.
// Modified files trigger a conflict prompt unless --force is set.
func Update(opts Options) error {
	dir := opts.dir()

	if err := checkDetached(dir, opts.Force); err != nil {
		return fmt.Errorf("harness update: %w", err)
	}

	projectName := opts.ProjectName
	if projectName == "" {
		projectName = DetectProjectName(dir)
	}

	w := opts.stdout()
	manifest := Manifest(projectName)

	for _, hf := range manifest {
		absPath := filepath.Join(dir, hf.RelPath)

		if hf.IsDir {
			if opts.DryRun {
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					fmt.Fprintf(w, "harness update: would create directory %s\n", hf.RelPath)
				}
				continue
			}
			if err := os.MkdirAll(absPath, 0o755); err != nil {
				return fmt.Errorf("harness update: failed to create directory %s: %w", hf.RelPath, err)
			}
			continue
		}

		// File entry.
		existing, err := os.ReadFile(absPath)
		if os.IsNotExist(err) {
			// File doesn't exist — create it.
			if opts.DryRun {
				fmt.Fprintf(w, "harness update: would create %s\n", hf.RelPath)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
				return fmt.Errorf("harness update: %w", err)
			}
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness update: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness update: created %s\n", hf.RelPath)
			continue
		} else if err != nil {
			return fmt.Errorf("harness update: failed to read %s: %w", hf.RelPath, err)
		}

		// File exists — compare content.
		if string(existing) == hf.Content {
			continue // Already up to date.
		}

		// Content differs — resolve conflict.
		if opts.DryRun {
			fmt.Fprintf(w, "harness update: would update %s (content differs)\n", hf.RelPath)
			continue
		}

		if opts.Force {
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness update: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness update: updated %s (--force)\n", hf.RelPath)
			continue
		}

		action, err := promptConflict(opts.stdin(), w, hf.RelPath)
		if err != nil {
			return fmt.Errorf("harness update: %w", err)
		}

		switch action {
		case ActionAbort:
			return fmt.Errorf("harness update: aborted by user at %s", hf.RelPath)
		case ActionOverwrite:
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness update: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness update: updated %s\n", hf.RelPath)
		case ActionSkip:
			fmt.Fprintf(w, "harness update: skipped %s\n", hf.RelPath)
		case ActionRenameAndReference:
			if err := renameAndReference(absPath, hf.RelPath, hf.Content, w); err != nil {
				return fmt.Errorf("harness update: %w", err)
			}
		}
	}

	return nil
}
