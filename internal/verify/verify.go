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
	// Base is a git ref. When set, policy judges every change since the
	// merge base of Base and HEAD, not just uncommitted work.
	Base string
}

// Run executes verification for the given spec.
func Run(mas *spec.MAS, opts Options) (*evidence.Report, error) {
	report := evidence.NewReport(opts.Profile)

	scope := runner.PolicyScope{}
	if opts.Base != "" {
		sha, err := gitrepo.MergeBase("", opts.Base)
		if err != nil {
			return nil, fmt.Errorf("--base %s: %w", opts.Base, err)
		}
		scope.BaseSHA = sha
		report.BaseSHA = sha
	}

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

	if mas.Policy != nil {
		report.Policy = runner.RunPolicy(mas.Policy, scope)
	}

	report.ComputeVerdict()
	return report, nil
}
