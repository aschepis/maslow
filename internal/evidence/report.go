// Package evidence handles creation and emission of verification reports.
package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Report is the top-level verification/audit report structure.
type Report struct {
	Timestamp    string           `json:"timestamp"`
	GitSHA       string           `json:"git_sha,omitempty"`
	BaseSHA      string           `json:"base_sha,omitempty"`
	Profile      string           `json:"profile"`
	Verdict      string           `json:"verdict"` // pass, fail, inconclusive
	CheckResults []CheckResult    `json:"check_results"`
	Contracts    []ContractResult `json:"contracts,omitempty"`
	Budgets      []BudgetResult   `json:"budgets,omitempty"`
	Refs         []RefResult      `json:"refs,omitempty"`
	Policy       []PolicyResult   `json:"policy,omitempty"`
}

// CheckResult records the outcome of a single check.
type CheckResult struct {
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Status   string  `json:"status"` // pass, fail, skip, error
	Duration float64 `json:"duration_seconds"`
	Output   string  `json:"output,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// ContractResult records the outcome of a contract scenario.
type ContractResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// BudgetResult records the outcome of a budget check.
type BudgetResult struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Actual string `json:"actual,omitempty"`
	Limit  string `json:"limit,omitempty"`
	Error  string `json:"error,omitempty"`
}

// RefResult records the outcome of a ref verification.
type RefResult struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Status string `json:"status"` // pass, fail, skip
	Error  string `json:"error,omitempty"`
}

// PolicyResult records the outcome of a policy rule check.
type PolicyResult struct {
	Rule   string   `json:"rule"`
	Kind   string   `json:"kind"` // deny, protected, gated
	Status string   `json:"status"`
	Detail string   `json:"detail,omitempty"`
	Files  []string `json:"files,omitempty"`
	Error  string   `json:"error,omitempty"`
}

// NewReport creates a report with timestamp and git SHA.
func NewReport(profile string) *Report {
	return &Report{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GitSHA:    gitSHA(),
		Profile:   profile,
		Verdict:   "inconclusive",
	}
}

// ComputeVerdict sets the verdict from every section of the report.
// Any fail or error gives "fail"; "inconclusive" means nothing ran at all.
func (r *Report) ComputeVerdict() {
	statuses := r.allStatuses()

	ran := false
	for _, s := range statuses {
		switch s {
		case "fail", "error":
			r.Verdict = "fail"
			return
		case "pass":
			ran = true
		}
	}

	if ran {
		r.Verdict = "pass"
	} else {
		r.Verdict = "inconclusive"
	}
}

func (r *Report) allStatuses() []string {
	var statuses []string
	for _, c := range r.CheckResults {
		statuses = append(statuses, c.Status)
	}
	for _, c := range r.Contracts {
		statuses = append(statuses, c.Status)
	}
	for _, b := range r.Budgets {
		statuses = append(statuses, b.Status)
	}
	for _, ref := range r.Refs {
		statuses = append(statuses, ref.Status)
	}
	for _, p := range r.Policy {
		statuses = append(statuses, p.Status)
	}
	return statuses
}

// Write emits the report as JSON to the given path.
func (r *Report) Write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write report to %s: %w", path, err)
	}

	return nil
}

func gitSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
