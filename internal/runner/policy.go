package runner

import (
	"fmt"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/spec"
)

// PolicyScope says which repository policy is judged in and against what.
// With an empty BaseSHA only uncommitted changes count as changes.
type PolicyScope struct {
	Dir     string
	BaseSHA string
}

// RunPolicy evaluates policy rules and returns results.
func RunPolicy(policy *spec.Policy, scope PolicyScope) []evidence.PolicyResult {
	if policy == nil {
		return nil
	}

	var results []evidence.PolicyResult

	if len(policy.Deny) > 0 {
		files, err := gitrepo.CandidateFiles(scope.Dir)
		if err != nil {
			results = append(results, ruleErrors("deny", policy.Deny, err)...)
		} else {
			results = append(results, checkDenyRules(policy.Deny, files)...)
		}
	}

	if len(policy.Protected) > 0 {
		changed, err := changedFiles(scope)
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

// changedFiles lists what a change touched: everything since the base commit
// when one is given, otherwise only uncommitted work.
func changedFiles(scope PolicyScope) ([]string, error) {
	if scope.BaseSHA == "" {
		return gitrepo.WorkingTreeChanges(scope.Dir)
	}
	return gitrepo.ChangedSince(scope.Dir, scope.BaseSHA)
}
