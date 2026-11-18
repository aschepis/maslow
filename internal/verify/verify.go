// Package verify implements the maslow verify command logic.
package verify

import (
	"errors"
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
	// SpecAddedOnBranch records that SpecFromBase was asked for but the base
	// had no spec, so the branch's own spec was used and needs review.
	SpecAddedOnBranch bool
	// RedGreen also runs overlay contracts on BaseSHA and passes only those
	// that fail there, so an overlay cannot pass without the change.
	RedGreen bool
}

// ReadSpec loads the spec opts asks for. When the spec comes from the base
// and the base has none, as on the change that adopts Maslow, it falls back
// to the branch's spec and sets SpecAddedOnBranch instead of failing.
func ReadSpec(opts *Options) (*spec.Loaded, error) {
	if !opts.SpecFromBase {
		return spec.Read(opts.SpecPath, "")
	}
	loaded, err := spec.Read(opts.SpecPath, opts.BaseSHA)
	if !errors.Is(err, spec.ErrNotAtCommit) {
		return loaded, err
	}
	opts.SpecFromBase = false
	opts.SpecAddedOnBranch = true
	return spec.Read(opts.SpecPath, "")
}

// SpecLoadFailure is the evidence for a run that could not load its spec from
// the base, so a pipeline reading the report sees why instead of nothing.
func SpecLoadFailure(opts Options, err error) *evidence.Report {
	report := evidence.NewReport(evidence.Meta{
		Command:    "verify",
		Profile:    opts.Profile,
		SpecPath:   opts.SpecPath,
		SpecSource: "base",
		BaseSHA:    opts.BaseSHA,
	})
	report.Policy = []evidence.PolicyResult{{Rule: opts.SpecPath, Kind: "spec", Status: "error", Error: err.Error()}}
	report.ComputeVerdict()
	return report
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
	if opts.RedGreen && opts.BaseSHA == "" {
		return nil, errors.New("--red-green needs --base to know which commit the overlay contracts must fail on")
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
	var base *baseTree
	if opts.RedGreen && len(opts.Overlays) > 0 {
		base = checkoutBase(opts.BaseSHA)
		defer base.remove()
	}
	for _, o := range opts.Overlays {
		report.Overlays = append(report.Overlays, evidence.OverlayRef{Path: o.Path, SHA256: o.SHA256})
		results := runner.RunContracts(o.Overlay.Contracts)
		if base != nil {
			judgeRedGreen(o.Overlay.Contracts, results, base)
		}
		for _, result := range results {
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
		BaseSHA:           opts.BaseSHA,
		SpecPath:          opts.SpecPath,
		SpecFromBase:      opts.SpecFromBase,
		SpecAddedOnBranch: opts.SpecAddedOnBranch,
	})

	report.ComputeVerdict()
	return report, nil
}

// enabled treats an unset profile switch as on.
func enabled(flag *bool) bool {
	return flag == nil || *flag
}
