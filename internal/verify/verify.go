// Package verify implements the maslow verify command logic.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/runner"
	"github.com/aschepis/maslow/internal/schema"
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
	SpecPath string
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

// LoadSpec reads, validates and parses the spec at path. With a non-empty
// atCommit the spec is read from that commit instead of the working tree.
func LoadSpec(path, atCommit string) (*spec.MAS, error) {
	data, source, err := readSpec(path, atCommit)
	if err != nil {
		return nil, err
	}

	result, err := schema.Validate(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	if !result.Valid {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("%s is invalid:\n  %s", source, strings.Join(msgs, "\n  "))
	}

	return spec.Parse(data, source)
}

func readSpec(path, atCommit string) ([]byte, string, error) {
	if atCommit == "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, path, fmt.Errorf("failed to read %s: %w", path, err)
		}
		return data, path, nil
	}

	if filepath.IsAbs(path) {
		return nil, path, fmt.Errorf("--spec must be a relative path when reading the spec from the base commit, got %s", path)
	}
	source := fmt.Sprintf("%s at %s", path, shortSHA(atCommit))
	data, err := gitrepo.Show("", atCommit, "./"+filepath.ToSlash(path))
	if err != nil {
		return nil, source, fmt.Errorf("failed to read %s: %w", source, err)
	}
	return data, source, nil
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// Run executes verification for the given spec.
func Run(mas *spec.MAS, opts Options) (*evidence.Report, error) {
	report := evidence.NewReport(opts.Profile)
	report.BaseSHA = opts.BaseSHA

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
