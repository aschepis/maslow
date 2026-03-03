package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// RunPolicy evaluates policy rules and returns results.
func RunPolicy(policy *spec.Policy) []evidence.PolicyResult {
	if policy == nil {
		return nil
	}

	var results []evidence.PolicyResult
	results = append(results, checkDenyRules(policy.Deny)...)
	results = append(results, checkProtectedFiles(policy.Protected)...)

	// Gated and allow rules are deferred to future versions.
	for _, g := range policy.Gated {
		results = append(results, evidence.PolicyResult{
			Rule:   g.Path,
			Kind:   "gated",
			Status: "skip",
			Detail: fmt.Sprintf("gated directory enforcement not yet implemented (owner: %s)", g.Owner),
		})
	}

	return results
}

func checkDenyRules(patterns []string) []evidence.PolicyResult {
	var results []evidence.PolicyResult
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   "deny",
				Status: "error",
				Error:  fmt.Sprintf("invalid glob pattern: %v", err),
			})
			continue
		}

		if len(matches) > 0 {
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   "deny",
				Status: "fail",
				Detail: fmt.Sprintf("denied pattern matches %d file(s): %s", len(matches), strings.Join(matches, ", ")),
			})
		} else {
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   "deny",
				Status: "pass",
			})
		}
	}
	return results
}

func checkProtectedFiles(paths []string) []evidence.PolicyResult {
	var results []evidence.PolicyResult
	for _, path := range paths {
		modified, err := isFileModified(path)
		if err != nil {
			results = append(results, evidence.PolicyResult{
				Rule:   path,
				Kind:   "protected",
				Status: "error",
				Error:  fmt.Sprintf("failed to check protected file: %v", err),
			})
			continue
		}

		if modified {
			results = append(results, evidence.PolicyResult{
				Rule:   path,
				Kind:   "protected",
				Status: "fail",
				Detail: fmt.Sprintf("protected file has uncommitted modifications: %s", path),
			})
		} else {
			results = append(results, evidence.PolicyResult{
				Rule:   path,
				Kind:   "protected",
				Status: "pass",
			})
		}
	}
	return results
}

// isFileModified checks if a file has uncommitted changes using git.
func isFileModified(path string) (bool, error) {
	// Check for staged or unstaged changes.
	out, err := exec.Command("git", "diff", "--name-only", "HEAD", "--", path).Output()
	if err != nil {
		// If git diff fails (e.g., file not tracked), fall back to status check.
		return isFileUntracked(path)
	}
	if strings.TrimSpace(string(out)) != "" {
		return true, nil
	}

	// Also check for untracked status (new file not yet committed).
	return isFileUntracked(path)
}

func isFileUntracked(path string) (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain", path).Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return false, nil
	}
	// Any porcelain output means the file is modified/untracked.
	return true, nil
}
