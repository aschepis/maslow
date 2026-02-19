// Package verify implements the maslow verify command logic.
package verify

import (
	"fmt"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/runner"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

const DefaultReportPath = "reports/verify.json"

// Run executes verification for the given spec and profile.
func Run(mas *spec.MAS, profile string) (*evidence.Report, error) {
	report := evidence.NewReport(profile)

	// Run checks for the profile.
	results, err := runner.RunProfileChecks(mas, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to run checks: %w", err)
	}
	report.CheckResults = results

	// Run contracts.
	if len(mas.Contracts) > 0 {
		report.Contracts = runner.RunContracts(mas.Contracts)
	}

	// Run budgets.
	if len(mas.Budgets) > 0 {
		report.Budgets = runner.RunBudgets(mas.Budgets)
	}

	report.ComputeVerdict()
	return report, nil
}
