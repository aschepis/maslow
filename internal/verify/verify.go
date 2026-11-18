// Package verify implements the maslow verify command logic.
package verify

import (
	"fmt"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/runner"
	"github.com/aschepis/maslow/internal/spec"
)

const DefaultReportPath = "reports/verify.json"

// Options controls a verification run.
type Options struct {
	Profile string
	// BaseSHA, when set, makes policy judge every change since that commit
	// rather than only uncommitted work.
	BaseSHA string
	// SpecPath is the spec's path relative to the repository root.
	SpecPath   string
	SpecSHA256 string
	// SpecFromBase records that the spec was read from BaseSHA, so a change
	// to the spec on this branch is reported instead of trusted.
	SpecFromBase bool
	// Overlays add contracts that run on every profile, beside those the profile selects.
	Overlays []*spec.LoadedOverlay
}

// ResolveBase returns the merge base of ref and HEAD.
func ResolveBase(ref string) (string, error) {
	sha, err := gitrepo.MergeBase("", ref)
	if err != nil {
		return "", fmt.Errorf("--base %s: %w", ref, err)
	}
	return sha, nil
}

// Run executes verification for the given spec.
func Run(mas *spec.MAS, opts Options) (*evidence.Report, error) {
	specSource := "head"
	if opts.SpecFromBase {
		specSource = "base"
	}
	report := evidence.NewReport(evidence.Meta{
		Command:    "verify",
		Profile:    opts.Profile,
		SpecPath:   opts.SpecPath,
		SpecSHA256: opts.SpecSHA256,
		SpecSource: specSource,
		BaseSHA:    opts.BaseSHA,
	})

	profile, ok := mas.Profiles[opts.Profile]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in spec", opts.Profile)
	}
	if err := spec.CheckOverlayNames(mas, opts.Overlays); err != nil {
		return nil, err
	}
	contracts, err := runner.SelectContracts(mas.Contracts, "profiles."+opts.Profile, profile.Contracts)
	if err != nil {
		return nil, err
	}
	budgets, err := runner.SelectBudgets(mas.Budgets, "profiles."+opts.Profile, profile.Budgets)
	if err != nil {
		return nil, err
	}

	results, err := runner.RunProfileChecks(mas, opts.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to run checks: %w", err)
	}
	report.CheckResults = results

	if len(contracts) > 0 {
		report.Contracts = runner.RunContracts(contracts)
	}
	for _, o := range opts.Overlays {
		report.Overlays = append(report.Overlays, evidence.OverlayRef{Path: o.Path, SHA256: o.SHA256})
		for _, result := range runner.RunContracts(o.Overlay.Contracts) {
			result.Overlay = o.Path
			report.Contracts = append(report.Contracts, result)
		}
	}

	if len(budgets) > 0 {
		report.Budgets = runner.RunBudgets(budgets)
	}

	if len(mas.Refs) > 0 && enabled(profile.Refs) {
		report.Refs = runner.RunRefs(mas.Refs)
	}

	// policy: false turns off the spec's rules, never the spec-change check.
	rules := mas.Policy
	if !enabled(profile.Policy) {
		rules = nil
	}
	report.Policy = runner.RunPolicy(rules, runner.PolicyScope{
		BaseSHA:      opts.BaseSHA,
		SpecPath:     opts.SpecPath,
		SpecFromBase: opts.SpecFromBase,
	})

	report.ComputeVerdict()
	return report, nil
}

// enabled treats an unset profile switch as on.
func enabled(flag *bool) bool {
	return flag == nil || *flag
}
