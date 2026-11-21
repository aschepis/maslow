// Package verify implements the maslow verify command logic.
package verify

import (
	"errors"
	"fmt"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/progress"
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
	// Progress, when set, hears each item start and finish. It never changes the evidence.
	Progress progress.Func
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
	if err := spec.CheckOverlayRetries(mas, opts.Overlays); err != nil {
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

	checks, err := runner.ProfileChecks(mas, opts.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to run checks: %w", err)
	}
	var refs []spec.Ref
	if enabled(profile.Refs) {
		refs = mas.Refs
	}
	emit := opts.Progress
	emit.Emit(progress.Event{Phase: progress.Plan, Total: planned(checks, contracts, opts.Overlays, budgets, refs)})

	if mas.Checks != nil {
		report.CheckResults = make([]evidence.CheckResult, 0, len(checks))
	}
	for _, check := range checks {
		emit.Step("check", check.Name, func() ([]string, int) {
			result := runner.RunChecks([]spec.CheckRunner{check})[0]
			report.CheckResults = append(report.CheckResults, result)
			return []string{result.Status}, max(1, len(result.Attempts))
		})
	}

	for _, contract := range contracts {
		emit.Step("contract", contract.Name, func() ([]string, int) {
			results := runner.RunContracts([]spec.Contract{contract})
			report.Contracts = append(report.Contracts, results...)
			return contractOutcome(results)
		})
	}
	var base *baseTree
	if opts.RedGreen && len(opts.Overlays) > 0 {
		base = checkoutBase(opts.BaseSHA)
		defer base.remove()
	}
	for _, o := range opts.Overlays {
		report.Overlays = append(report.Overlays, evidence.OverlayRef{Path: o.Path, SHA256: o.SHA256})
		for _, contract := range o.Overlay.Contracts {
			emit.Step("contract", contract.Name, func() ([]string, int) {
				one := []spec.Contract{contract}
				results := runner.RunContracts(one)
				if base != nil {
					judgeRedGreen(one, results, base)
				}
				for _, result := range results {
					result.Overlay = o.Path
					report.Contracts = append(report.Contracts, result)
				}
				return contractOutcome(results)
			})
		}
	}

	for _, budget := range budgets {
		emit.Step("budget", budget.Name, func() ([]string, int) {
			result := runner.RunBudgets([]spec.Budget{budget})[0]
			report.Budgets = append(report.Budgets, result)
			return []string{result.Status}, 1
		})
	}

	for _, ref := range refs {
		emit.Step("ref", runner.RefPath(ref), func() ([]string, int) {
			result := runner.RunRefs([]spec.Ref{ref})[0]
			report.Refs = append(report.Refs, result)
			return []string{result.Status}, 1
		})
	}

	// policy: false turns off the spec's rules, never the spec-change check.
	rules := mas.Policy
	if !enabled(profile.Policy) {
		rules = nil
	}
	emit.Step("policy", "", func() ([]string, int) {
		report.Policy = runner.RunPolicy(rules, runner.PolicyScope{
			BaseSHA:           opts.BaseSHA,
			SpecPath:          opts.SpecPath,
			SpecFromBase:      opts.SpecFromBase,
			SpecAddedOnBranch: opts.SpecAddedOnBranch,
		})
		var statuses []string
		for _, p := range report.Policy {
			statuses = append(statuses, p.Status)
		}
		return statuses, 1
	})

	report.ComputeVerdict()
	report.ComputeCoverage()
	return report, nil
}

// planned counts the items a run reports: each check, contract, budget and
// ref, and the policy rules as one.
func planned(checks []spec.CheckRunner, contracts []spec.Contract, overlays []*spec.LoadedOverlay, budgets []spec.Budget, refs []spec.Ref) int {
	total := len(checks) + len(contracts) + len(budgets) + len(refs) + 1
	for _, o := range overlays {
		total += len(o.Overlay.Contracts)
	}
	return total
}

// contractOutcome is each scenario's status and the most attempts any scenario took.
func contractOutcome(results []evidence.ContractResult) ([]string, int) {
	statuses := make([]string, 0, len(results))
	attempts := 1
	for _, r := range results {
		statuses = append(statuses, r.Status)
		attempts = max(attempts, len(r.Attempts))
	}
	return statuses, attempts
}

// enabled treats an unset profile switch as on.
func enabled(flag *bool) bool {
	return flag == nil || *flag
}
