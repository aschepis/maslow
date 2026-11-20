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
	Overlays        []OverlayRef     `json:"overlays,omitempty"`
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

// OverlayRef records which overlay bytes added contracts to a run.
type OverlayRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
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
	// Attempts is set only when the check ran more than once; the fields
	// above then describe the last attempt, and Duration is the total.
	Attempts []CheckAttempt `json:"attempts,omitempty"`
}

// StatusFlaky marks a check or contract that failed and then passed on a retry.
const StatusFlaky = "flaky"

// CheckAttempt is one run of a retried check.
type CheckAttempt struct {
	Status          string  `json:"status"`
	Duration        float64 `json:"duration_seconds"`
	Output          string  `json:"output,omitempty"`
	OutputBytes     int     `json:"output_bytes,omitempty"`
	OutputTruncated bool    `json:"output_truncated,omitempty"`
	Error           string  `json:"error,omitempty"`
}

// ContractAttempt is one run of a retried contract scenario.
type ContractAttempt struct {
	Status          string        `json:"status"`
	Error           string        `json:"error,omitempty"`
	FailedStep      *FailedStep   `json:"failed_step,omitempty"`
	Output          string        `json:"output,omitempty"`
	OutputBytes     int           `json:"output_bytes,omitempty"`
	OutputTruncated bool          `json:"output_truncated,omitempty"`
	Artifacts       []string      `json:"artifacts,omitempty"`
	Observations    []Observation `json:"observations,omitempty"`
}

// Observation records what one metric, log or alerts step asked and saw, so
// a reviewer can replay each query at its recorded time.
type Observation struct {
	Phase    string `json:"phase"`
	Index    int    `json:"index"`
	Action   string `json:"action"`
	Provider string `json:"provider"`
	// URL is as written in the spec, before templating.
	URL               string            `json:"url"`
	Query             string            `json:"query,omitempty"`
	Matchers          []string          `json:"matchers,omitempty"`
	IgnorePreexisting bool              `json:"ignore_preexisting,omitempty"`
	For               string            `json:"for,omitempty"`
	Interval          string            `json:"interval,omitempty"`
	Settle            string            `json:"settle,omitempty"`
	Expect            ObservationBounds `json:"expect"`
	Samples           []Sample          `json:"samples"`
	// Breach says why the step failed, naming the first offending series or alert.
	Breach string `json:"breach,omitempty"`
}

// ObservationBounds are the inclusive bounds a step judged its samples by.
type ObservationBounds struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinSeries *int     `json:"min_series,omitempty"`
	MaxSeries *int     `json:"max_series,omitempty"`
	IfNoData  string   `json:"if_no_data,omitempty"`
}

// Sample is one query at one evaluation time. Min and Max span every series
// and are strings so that +Inf and -Inf survive JSON.
type Sample struct {
	At     string `json:"at"`
	Series *int   `json:"series,omitempty"`
	Alerts *int   `json:"alerts,omitempty"`
	Min    string `json:"min,omitempty"`
	Max    string `json:"max,omitempty"`
	NoData bool   `json:"no_data,omitempty"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

// ContractResult records the outcome of a contract scenario.
type ContractResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// Overlay is the path of the overlay that added the contract, if any.
	Overlay    string      `json:"overlay,omitempty"`
	FailedStep *FailedStep `json:"failed_step,omitempty"`
	// Output is what the step before the failure printed or returned.
	Output          string `json:"output,omitempty"`
	OutputBytes     int    `json:"output_bytes,omitempty"`
	OutputTruncated bool   `json:"output_truncated,omitempty"`
	// Artifacts are files matching the scenario's artifact patterns, such as screenshots.
	Artifacts []string `json:"artifacts,omitempty"`
	// Observations has one entry per metric, log or alerts step that ran.
	Observations []Observation `json:"observations,omitempty"`
	// Breached marks a failure on what the system reported. Rerunning would
	// cost another hold and could relabel a real breach as flaky.
	Breached bool `json:"-"`
	// HeadStatus and Base are set only by a red-green run, where Status
	// combines the two runs.
	HeadStatus string   `json:"head_status,omitempty"`
	Base       *BaseRun `json:"base,omitempty"`
	// Attempts is set only when the scenario ran more than once; the fields
	// above then describe the last attempt.
	Attempts []ContractAttempt `json:"attempts,omitempty"`
}

// StatusNotRed marks an overlay contract that already passed on the base
// commit, so its pass on the head says nothing about the change.
const StatusNotRed = "not_red"

// BaseRun is the outcome of an overlay contract run against the base commit.
type BaseRun struct {
	SHA        string            `json:"sha"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	FailedStep *FailedStep       `json:"failed_step,omitempty"`
	Attempts   []ContractAttempt `json:"attempts,omitempty"`
}

// FailedStep identifies the step that failed a contract scenario.
type FailedStep struct {
	Phase  string `json:"phase"` // contract_setup, setup, steps
	Index  int    `json:"index"` // zero-based position within the phase
	Action string `json:"action"`
	Target string `json:"target,omitempty"` // command line or "METHOD url", after substitution
	// PageURL and Console describe the page a failing browser step left behind.
	PageURL string   `json:"page_url,omitempty"`
	Console []string `json:"console,omitempty"`
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

// ModeObserve marks a policy result that is reported but does not affect the verdict.
const ModeObserve = "observe"

// PolicyResult records the outcome of a policy rule check.
type PolicyResult struct {
	Rule string `json:"rule"`
	Kind string `json:"kind"` // deny, allow, protected, gated, spec, shrink_only, test_integrity
	// Mode "observe" marks a rule that is reported but not enforced; its
	// non-pass statuses are prefixed "would_".
	Mode   string   `json:"mode,omitempty"`
	Status string   `json:"status"`
	Detail string   `json:"detail,omitempty"`
	Files  []string `json:"files,omitempty"`
	// Lines are the "path:line: text" entries a line-level rule flagged.
	Lines []string `json:"lines,omitempty"`
	Owner string   `json:"owner,omitempty"` // who reviews a gated change
	Error string   `json:"error,omitempty"`
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

// CapOutput keeps at most limit bytes of each check's and contract's output,
// keeping the tail where failures are usually reported. A limit <= 0 keeps everything.
func (r *Report) CapOutput(limit int) {
	if limit <= 0 {
		return
	}
	for i := range r.CheckResults {
		c := &r.CheckResults[i]
		capOutput(&c.Output, &c.OutputBytes, &c.OutputTruncated, limit)
		for j := range c.Attempts {
			a := &c.Attempts[j]
			capOutput(&a.Output, &a.OutputBytes, &a.OutputTruncated, limit)
		}
	}
	for i := range r.Contracts {
		c := &r.Contracts[i]
		capOutput(&c.Output, &c.OutputBytes, &c.OutputTruncated, limit)
		capContractAttempts(c.Attempts, limit)
		if c.Base != nil {
			capContractAttempts(c.Base.Attempts, limit)
		}
	}
}

func capContractAttempts(attempts []ContractAttempt, limit int) {
	for i := range attempts {
		a := &attempts[i]
		capOutput(&a.Output, &a.OutputBytes, &a.OutputTruncated, limit)
	}
}

func capOutput(output *string, fullBytes *int, truncated *bool, limit int) {
	if len(*output) <= limit {
		return
	}
	*fullBytes = len(*output)
	*truncated = true
	*output = tail(*output, limit)
}

// tail returns the last n bytes of s, starting on a rune boundary.
func tail(s string, n int) string {
	start := len(s) - n
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

// ComputeVerdict sets the verdict from every section of the report, with
// precedence fail, needs_review, inconclusive, pass. Any fail, error or
// not_red gives "fail"; "inconclusive" means nothing ran at all. Observed policy rules never count,
// and a flaky result counts as a pass unless it is an overlay contract (ADR 0039).
func (r *Report) ComputeVerdict() {
	ran, review := false, false
	for _, s := range r.allStatuses() {
		switch s {
		case "fail", "error", StatusNotRed:
			r.Verdict = "fail"
			return
		case "needs_review":
			review = true
		case "pass", StatusFlaky:
			ran = true
		}
	}

	switch {
	case review:
		r.Verdict = "needs_review"
	case ran:
		r.Verdict = "pass"
	default:
		r.Verdict = "inconclusive"
	}
}

func (r *Report) allStatuses() []string {
	var statuses []string
	for _, c := range r.CheckResults {
		statuses = append(statuses, c.Status)
	}
	for _, c := range r.Contracts {
		statuses = append(statuses, contractVerdictStatus(c))
	}
	for _, b := range r.Budgets {
		statuses = append(statuses, b.Status)
	}
	for _, ref := range r.Refs {
		statuses = append(statuses, ref.Status)
	}
	for _, p := range r.Policy {
		if p.Mode == ModeObserve {
			continue
		}
		statuses = append(statuses, p.Status)
	}
	return statuses
}

// contractVerdictStatus asks for review of a flaky overlay contract, because
// an acceptance contract is where a flaky fix would hide.
func contractVerdictStatus(c ContractResult) string {
	if c.Status == StatusFlaky && c.Overlay != "" {
		return "needs_review"
	}
	return c.Status
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
