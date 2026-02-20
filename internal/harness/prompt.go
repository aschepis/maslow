package harness

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// promptConflict asks the user how to resolve a file conflict.
// Returns ActionAbort on EOF or unrecoverable read error.
func promptConflict(r io.Reader, w io.Writer, path string) (ConflictAction, error) {
	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprintf(w, "harness: conflict: %s already exists\n", path)
		fmt.Fprintf(w, "  [a]bort  [o]verwrite  [s]kip  [r]ename-and-reference\n")
		fmt.Fprintf(w, "  Choice: ")

		if !scanner.Scan() {
			// EOF or read error — treat as abort.
			return ActionAbort, nil
		}

		choice := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch choice {
		case "a", "abort":
			return ActionAbort, nil
		case "o", "overwrite":
			return ActionOverwrite, nil
		case "s", "skip":
			return ActionSkip, nil
		case "r", "rename-and-reference", "rename":
			return ActionRenameAndReference, nil
		default:
			fmt.Fprintf(w, "  Invalid choice %q, try again.\n", choice)
		}
	}
}
