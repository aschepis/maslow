package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// RunPolicy evaluates policy rules against the repository in the current
// directory and returns results.
func RunPolicy(policy *spec.Policy) []evidence.PolicyResult {
	if policy == nil {
		return nil
	}

	var results []evidence.PolicyResult

	if len(policy.Deny) > 0 {
		files, err := candidateFiles("")
		if err != nil {
			results = append(results, ruleErrors("deny", policy.Deny, err)...)
		} else {
			results = append(results, checkDenyRules(policy.Deny, files)...)
		}
	}

	if len(policy.Protected) > 0 {
		changed, err := workingTreeChanges("")
		if err != nil {
			results = append(results, ruleErrors("protected", policy.Protected, err)...)
		} else {
			results = append(results, checkProtectedFiles(policy.Protected, changed)...)
		}
	}

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

func checkDenyRules(patterns, files []string) []evidence.PolicyResult {
	return checkRules("deny", patterns, files, "denied pattern matches")
}

func checkProtectedFiles(patterns, changed []string) []evidence.PolicyResult {
	return checkRules("protected", patterns, changed, "protected path changed")
}

func checkRules(kind string, patterns, files []string, failPrefix string) []evidence.PolicyResult {
	results := make([]evidence.PolicyResult, 0, len(patterns))
	for _, pattern := range patterns {
		matched, err := matchingFiles(pattern, files)
		switch {
		case err != nil:
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   kind,
				Status: "error",
				Error:  fmt.Sprintf("invalid pattern: %v", err),
			})
		case len(matched) > 0:
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   kind,
				Status: "fail",
				Detail: fmt.Sprintf("%s %d file(s): %s", failPrefix, len(matched), strings.Join(matched, ", ")),
				Files:  matched,
			})
		default:
			results = append(results, evidence.PolicyResult{
				Rule:   pattern,
				Kind:   kind,
				Status: "pass",
			})
		}
	}
	return results
}

func ruleErrors(kind string, patterns []string, err error) []evidence.PolicyResult {
	results := make([]evidence.PolicyResult, 0, len(patterns))
	for _, pattern := range patterns {
		results = append(results, evidence.PolicyResult{
			Rule:   pattern,
			Kind:   kind,
			Status: "error",
			Error:  err.Error(),
		})
	}
	return results
}

// candidateFiles lists the files a deny rule applies to: tracked files plus
// untracked files that are not ignored, so a local, gitignored .env never fails.
func candidateFiles(dir string) ([]string, error) {
	out, err := git(dir, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return splitNul(out), nil
}

// workingTreeChanges lists files with staged, unstaged or untracked changes
// relative to HEAD, excluding ignored files.
func workingTreeChanges(dir string) ([]string, error) {
	out, err := git(dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}

	var changed []string
	entries := splitNul(out)
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue
		}
		changed = append(changed, entry[3:])
		// A rename or copy entry is followed by its source path.
		if entry[0] == 'R' || entry[0] == 'C' {
			i++
			if i < len(entries) {
				changed = append(changed, entries[i])
			}
		}
	}
	sort.Strings(changed)
	return changed, nil
}

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func splitNul(out []byte) []string {
	var parts []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
