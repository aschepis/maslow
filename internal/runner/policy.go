package runner

import (
	"fmt"
	"path/filepath"
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
	// SpecPath and SpecFromBase report a spec edited on this branch when the
	// run verified against the base commit's spec.
	SpecPath     string
	SpecFromBase bool
	// SpecAddedOnBranch reports that the base had no spec, so the branch's
	// own spec was verified and a person must accept it.
	SpecAddedOnBranch bool
}

// RunPolicy evaluates policy rules and returns results.
func RunPolicy(policy *spec.Policy, scope PolicyScope) []evidence.PolicyResult {
	var results []evidence.PolicyResult

	switch {
	case scope.SpecAddedOnBranch:
		results = append(results, evidence.PolicyResult{
			Rule:   scope.SpecPath,
			Kind:   "spec",
			Status: "needs_review",
			Detail: fmt.Sprintf("%s does not exist at base %s; this run verified against the branch's own spec, and a person must accept it", scope.SpecPath, scope.BaseSHA),
			Files:  []string{scope.SpecPath},
		})
	case scope.SpecFromBase && scope.BaseSHA != "":
		results = append(results, checkSpecUnchanged(scope))
	}

	if policy == nil {
		return results
	}

	results = append(results, runRules(policy, scope)...)
	if policy.Observe != nil {
		results = append(results, observed(runRules(policy.Observe, scope))...)
	}
	return results
}

// observedStatus names what enforcement would have done; pass and skip read the same either way.
var observedStatus = map[string]string{
	"fail":         "would_fail",
	"needs_review": "would_need_review",
	"error":        "would_error",
}

// observed marks results from observe-mode rules after evaluation, so every
// rule kind gets observe mode without knowing about it.
func observed(results []evidence.PolicyResult) []evidence.PolicyResult {
	for i := range results {
		results[i].Mode = evidence.ModeObserve
		if status, ok := observedStatus[results[i].Status]; ok {
			results[i].Status = status
		}
	}
	return results
}

// runRules evaluates one set of rules; it ignores policy.Observe.
func runRules(policy *spec.Policy, scope PolicyScope) []evidence.PolicyResult {
	var results []evidence.PolicyResult

	if len(policy.Deny) > 0 {
		files, err := gitrepo.CandidateFiles(scope.Dir)
		if err != nil {
			results = append(results, ruleErrors("deny", policy.Deny, err)...)
		} else {
			results = append(results, checkDenyRules(policy.Deny, files)...)
		}
	}

	results = append(results, runLineRules(policy, scope)...)

	if len(policy.Protected) == 0 && len(policy.Gated) == 0 && len(policy.Allow) == 0 {
		return results
	}

	changed, err := changedFiles(scope)
	if err != nil {
		results = append(results, ruleErrors("protected", policy.Protected, err)...)
		for _, g := range policy.Gated {
			results = append(results, evidence.PolicyResult{Rule: g.Path, Kind: "gated", Owner: g.Owner, Status: "error", Error: err.Error()})
		}
		if len(policy.Allow) > 0 {
			results = append(results, evidence.PolicyResult{Rule: strings.Join(policy.Allow, ", "), Kind: "allow", Status: "error", Error: err.Error()})
		}
		return results
	}

	results = append(results, checkProtectedFiles(policy.Protected, changed)...)
	results = append(results, checkGated(policy.Gated, changed)...)
	if len(policy.Allow) > 0 {
		results = append(results, checkAllow(policy.Allow, changed))
	}
	return results
}

// checkGated asks for review, rather than failing, when a change touches a
// path that has an owner.
func checkGated(gated []spec.GatedDir, changed []string) []evidence.PolicyResult {
	results := make([]evidence.PolicyResult, 0, len(gated))
	for _, g := range gated {
		result := evidence.PolicyResult{Rule: g.Path, Kind: "gated", Owner: g.Owner}
		matched, err := matchingFiles(g.Path, changed)
		switch {
		case err != nil:
			result.Status = "error"
			result.Error = fmt.Sprintf("invalid pattern: %v", err)
		case len(matched) > 0:
			result.Status = "needs_review"
			result.Detail = fmt.Sprintf("gated path changed %d file(s); review by %s: %s", len(matched), g.Owner, strings.Join(matched, ", "))
			result.Files = matched
		default:
			result.Status = "pass"
		}
		results = append(results, result)
	}
	return results
}

// checkAllow fails when a change touches a file outside every allowed path.
func checkAllow(allow, changed []string) evidence.PolicyResult {
	result := evidence.PolicyResult{Rule: strings.Join(allow, ", "), Kind: "allow"}

	var outside []string
	for _, f := range changed {
		inside := false
		for _, pattern := range allow {
			ok, err := matchPath(pattern, f)
			if err != nil {
				result.Status = "error"
				result.Error = fmt.Sprintf("invalid pattern %q: %v", pattern, err)
				return result
			}
			if ok {
				inside = true
				break
			}
		}
		if !inside {
			outside = append(outside, f)
		}
	}

	if len(outside) == 0 {
		result.Status = "pass"
		return result
	}
	result.Status = "fail"
	result.Detail = fmt.Sprintf("%d changed file(s) outside the allowed paths: %s", len(outside), strings.Join(outside, ", "))
	result.Files = outside
	return result
}

func checkSpecUnchanged(scope PolicyScope) evidence.PolicyResult {
	result := evidence.PolicyResult{Rule: scope.SpecPath, Kind: "spec"}

	changed, err := gitrepo.ChangedSince(scope.Dir, scope.BaseSHA)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	prefix, err := gitrepo.Prefix(scope.Dir)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}
	specPath := prefix + filepath.ToSlash(filepath.Clean(scope.SpecPath))
	for _, f := range changed {
		if f == specPath {
			result.Status = "needs_review"
			result.Detail = fmt.Sprintf("%s changed since base %s; this run verified against the base version, and a person must accept the new spec", specPath, scope.BaseSHA)
			result.Files = []string{f}
			return result
		}
	}

	result.Status = "pass"
	return result
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
