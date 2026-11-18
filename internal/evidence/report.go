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
	"unicode/utf8"

	"github.com/aschepis/maslow/internal/buildinfo"
)

// Version is the evidence format version; schema/evidence.schema.json describes it.
const Version = "2"

// DefaultOutputLimit is the default number of bytes of check output kept.
const DefaultOutputLimit = 4096

// Report is the top-level verification/audit report structure.
type Report struct {
	EvidenceVersion string           `json:"evidence_version"`
	MaslowVersion   string           `json:"maslow_version"`
	Command         string           `json:"command"` // verify, audit
	Timestamp       string           `json:"timestamp"`
	SpecPath        string           `json:"spec_path"`
	SpecSHA256      string           `json:"spec_sha256"`
	SpecSource      string           `json:"spec_source"` // head, base
	GitSHA          string           `json:"git_sha,omitempty"`
	BaseSHA         string           `json:"base_sha,omitempty"`
	Profile         string           `json:"profile"`
	Verdict         string           `json:"verdict"` // pass, fail, inconclusive
	CheckResults    []CheckResult    `json:"check_results"`
	Contracts       []ContractResult `json:"contracts,omitempty"`
	Budgets         []BudgetResult   `json:"budgets,omitempty"`
	Refs            []RefResult      `json:"refs,omitempty"`
	Policy          []PolicyResult   `json:"policy,omitempty"`
}

// Meta describes what a report is about.
type Meta struct {
	Command    string
	Profile    string
	SpecPath   string
	SpecSHA256 string
	SpecSource string
	BaseSHA    string
}

// CheckResult records the outcome of a single check.
type CheckResult struct {
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Status   string  `json:"status"` // pass, fail, skip, error
	Duration float64 `json:"duration_seconds"`
	Output   string  `json:"output,omitempty"`
	// OutputBytes is the size of the full output when Output was truncated.
	OutputBytes     int    `json:"output_bytes,omitempty"`
	OutputTruncated bool   `json:"output_truncated,omitempty"`
	Error           string `json:"error,omitempty"`
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

// NewReport creates a report stamped with the format version, the Maslow
// version, the time and the current git SHA.
func NewReport(meta Meta) *Report {
	return &Report{
		EvidenceVersion: Version,
		MaslowVersion:   buildinfo.Version(),
		Command:         meta.Command,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		SpecPath:        meta.SpecPath,
		SpecSHA256:      meta.SpecSHA256,
		SpecSource:      meta.SpecSource,
		GitSHA:          gitSHA(),
		BaseSHA:         meta.BaseSHA,
		Profile:         meta.Profile,
		Verdict:         "inconclusive",
	}
}

// CapOutput keeps at most limit bytes of each check's output, keeping the
// tail where failures are usually reported. A limit <= 0 keeps everything.
func (r *Report) CapOutput(limit int) {
	if limit <= 0 {
		return
	}
	for i := range r.CheckResults {
		c := &r.CheckResults[i]
		if len(c.Output) <= limit {
			continue
		}
		c.OutputBytes = len(c.Output)
		c.OutputTruncated = true
		c.Output = tail(c.Output, limit)
	}
}

// tail returns the last n bytes of s, starting on a rune boundary.
func tail(s string, n int) string {
	start := len(s) - n
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
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
