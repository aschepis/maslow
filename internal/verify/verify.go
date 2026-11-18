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

	results, err := runner.RunProfileChecks(mas, opts.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to run checks: %w", err)
	}
	report.CheckResults = results

	if len(mas.Contracts) > 0 {
		report.Contracts = runner.RunContracts(mas.Contracts)
	}

	if len(mas.Budgets) > 0 {
		report.Budgets = runner.RunBudgets(mas.Budgets)
	}

	if len(mas.Refs) > 0 {
		report.Refs = runner.RunRefs(mas.Refs)
	}

	report.Policy = runner.RunPolicy(mas.Policy, runner.PolicyScope{
		BaseSHA:      opts.BaseSHA,
		SpecPath:     opts.SpecPath,
		SpecFromBase: opts.SpecFromBase,
	})

	report.ComputeVerdict()
	return report, nil
}
