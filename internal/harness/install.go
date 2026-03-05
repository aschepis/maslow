package harness

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Install installs the agentic harness into a project directory.
// It creates missing files and prompts (or forces) on conflicts.
func Install(opts Options) error {
	dir := opts.dir()

	if err := checkDetached(dir, opts.Force); err != nil {
		return fmt.Errorf("harness install: %w", err)
	}

	// If force-installing over a detached harness, remove the sentinel.
	if opts.Force {
		os.Remove(filepath.Join(dir, SentinelFile))
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
					fmt.Fprintf(w, "harness install: would create directory %s\n", hf.RelPath)
				}
				continue
			}
			if err := os.MkdirAll(absPath, 0o755); err != nil {
				return fmt.Errorf("harness install: failed to create directory %s: %w", hf.RelPath, err)
			}
			continue
		}

		// File entry.
		_, err := os.Stat(absPath)
		exists := err == nil

		if !exists {
			if opts.DryRun {
				fmt.Fprintf(w, "harness install: would create %s\n", hf.RelPath)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
				return fmt.Errorf("harness install: %w", err)
			}
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness install: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness install: created %s\n", hf.RelPath)
			continue
		}

		// File exists — resolve conflict.
		if opts.DryRun {
			fmt.Fprintf(w, "harness install: would conflict %s (exists)\n", hf.RelPath)
			continue
		}

		if opts.Force {
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness install: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness install: overwrote %s (--force)\n", hf.RelPath)
			continue
		}

		action, err := promptConflict(opts.stdin(), w, hf.RelPath)
		if err != nil {
			return fmt.Errorf("harness install: %w", err)
		}

		switch action {
		case ActionAbort:
			return fmt.Errorf("harness install: aborted by user at %s", hf.RelPath)
		case ActionOverwrite:
			if err := os.WriteFile(absPath, []byte(hf.Content), 0o644); err != nil {
				return fmt.Errorf("harness install: failed to write %s: %w", hf.RelPath, err)
			}
			fmt.Fprintf(w, "harness install: overwrote %s\n", hf.RelPath)
		case ActionSkip:
			fmt.Fprintf(w, "harness install: skipped %s\n", hf.RelPath)
		case ActionRenameAndReference:
			if err := renameAndReference(absPath, hf.RelPath, hf.Content, w); err != nil {
				return fmt.Errorf("harness install: %w", err)
			}
		}
	}

	return nil
}

// renameAndReference renames the existing file to <name>.original and writes
// the new content with a reference note pointing to the original.
func renameAndReference(absPath, relPath, newContent string, w io.Writer) error {
	origPath := absPath + ".original"
	if err := os.Rename(absPath, origPath); err != nil {
		return fmt.Errorf("failed to rename %s: %w", relPath, err)
	}
	fmt.Fprintf(w, "harness install: renamed %s -> %s.original\n", relPath, relPath)

	// Prepend a reference note to the new content.
	note := fmt.Sprintf("<!-- NOTE: Your original %s has been preserved as %s.original -->\n\n",
		filepath.Base(relPath), filepath.Base(relPath))

	// For non-markdown files, use a comment-style note.
	if !strings.HasSuffix(relPath, ".md") {
		note = fmt.Sprintf("# NOTE: Your original %s has been preserved as %s.original\n\n",
			filepath.Base(relPath), filepath.Base(relPath))
	}

	if err := os.WriteFile(absPath, []byte(note+newContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", relPath, err)
	}
	fmt.Fprintf(w, "harness install: created %s (with reference to original)\n", relPath)
	return nil
}
