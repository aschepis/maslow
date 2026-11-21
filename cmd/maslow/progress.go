package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aschepis/maslow/internal/progress"
)

const progressUsage = "show each item as it runs on stderr: auto (when stderr is a terminal), always or never"

// progressFor returns the printer --progress asks for, or nil when nothing should print.
func progressFor(mode string, stderr *os.File) (progress.Func, error) {
	switch mode {
	case "always":
		return newProgressPrinter(stderr), nil
	case "never":
		return nil, nil
	case "auto":
		if isTerminal(stderr) {
			return newProgressPrinter(stderr), nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("--progress must be auto, always or never, got %q", mode)
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// newProgressPrinter writes one line when an item starts and one when it
// finishes, numbered against the planned total.
func newProgressPrinter(w io.Writer) progress.Func {
	var total, done int
	return func(e progress.Event) {
		label := strings.TrimSpace(e.Kind + " " + e.Name)
		switch e.Phase {
		case progress.Plan:
			total = e.Total
		case progress.Start:
			fmt.Fprintf(w, "[%d/%d] %s …\n", done+1, total, label)
		case progress.Finish:
			done++
			fmt.Fprintf(w, "[%d/%d] %s %s (%s)\n", done, total, label, summarizeStatuses(e.Statuses), timing(e))
		}
	}
}

// summarizeStatuses names a lone status as is and counts several, in the
// order each status first appears.
func summarizeStatuses(statuses []string) string {
	switch len(statuses) {
	case 0:
		return "done"
	case 1:
		return statuses[0]
	}
	counts := map[string]int{}
	var order []string
	for _, s := range statuses {
		if counts[s] == 0 {
			order = append(order, s)
		}
		counts[s]++
	}
	parts := make([]string, len(order))
	for i, s := range order {
		parts[i] = fmt.Sprintf("%d %s", counts[s], s)
	}
	return strings.Join(parts, ", ")
}

func timing(e progress.Event) string {
	elapsed := fmt.Sprintf("%.1fs", e.Duration.Seconds())
	if e.Attempts > 1 {
		return fmt.Sprintf("%d attempts, %s", e.Attempts, elapsed)
	}
	return elapsed
}
