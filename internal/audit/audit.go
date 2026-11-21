// Package audit implements the maslow audit command logic.
// Audit runs in black-box mode against built artifacts or endpoints.
package audit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/progress"
	"github.com/aschepis/maslow/internal/runner"
	"github.com/aschepis/maslow/internal/spec"
)

const DefaultReportPath = "reports/audit.json"

// Options controls an audit run.
type Options struct {
	Profile    string
	SpecPath   string
	SpecSHA256 string
	// Progress, when set, hears each target start and finish. It never changes the evidence.
	Progress progress.Func
}

// Run executes audit checks against the configured targets.
func Run(mas *spec.MAS, opts Options) (*evidence.Report, error) {
	report := evidence.NewReport(evidence.Meta{
		Command:    "audit",
		Profile:    opts.Profile,
		SpecPath:   opts.SpecPath,
		SpecSHA256: opts.SpecSHA256,
		SpecSource: "head",
	})

	if mas.Audit == nil || len(mas.Audit.Targets) == 0 {
		report.Verdict = "inconclusive"
		return report, nil
	}

	emit := opts.Progress
	emit.Emit(progress.Event{Phase: progress.Plan, Total: len(mas.Audit.Targets)})
	for _, target := range mas.Audit.Targets {
		emit.Step(target.Kind, target.Path, func() ([]string, int) {
			var checks []evidence.CheckResult
			var contracts []evidence.ContractResult
			if target.Kind == "endpoint" {
				checks, contracts = auditEndpoint(target, mas.Contracts)
			} else {
				checks = auditTarget(target)
			}
			report.CheckResults = append(report.CheckResults, checks...)
			report.Contracts = append(report.Contracts, contracts...)
			return targetOutcome(checks, contracts)
		})
	}

	report.ComputeVerdict()
	report.ComputeCoverage()
	return report, nil
}

// targetOutcome is the status of every result a target produced and the most
// attempts any of them took.
func targetOutcome(checks []evidence.CheckResult, contracts []evidence.ContractResult) ([]string, int) {
	var statuses []string
	attempts := 1
	for _, c := range checks {
		statuses = append(statuses, c.Status)
		attempts = max(attempts, len(c.Attempts))
	}
	for _, c := range contracts {
		statuses = append(statuses, c.Status)
		attempts = max(attempts, len(c.Attempts))
	}
	return statuses, attempts
}

func auditTarget(target spec.AuditTarget) []evidence.CheckResult {
	switch target.Kind {
	case "binary":
		return auditBinary(target)
	case "container":
		return auditContainer(target)
	default:
		return []evidence.CheckResult{{
			Name:   target.Path,
			Kind:   target.Kind,
			Status: "error",
			Error:  fmt.Sprintf("unsupported audit target kind: %s", target.Kind),
		}}
	}
}

func auditBinary(target spec.AuditTarget) []evidence.CheckResult {
	var results []evidence.CheckResult

	// Check binary exists.
	if _, err := os.Stat(target.Path); err != nil {
		results = append(results, evidence.CheckResult{
			Name:   target.Path + "/exists",
			Kind:   "binary",
			Status: "fail",
			Error:  fmt.Sprintf("binary not found: %s", target.Path),
		})
		return results
	}

	results = append(results, evidence.CheckResult{
		Name:   target.Path + "/exists",
		Kind:   "binary",
		Status: "pass",
	})

	// Run version check if configured.
	for _, check := range target.Checks {
		if check == "version-check" {
			result := runVersionCheck(target)
			results = append(results, result)
		}
	}

	return results
}

func runVersionCheck(target spec.AuditTarget) evidence.CheckResult {
	timeout := 30 * time.Second
	if target.Timeout != "" {
		if d, err := parseDuration(target.Timeout); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, target.Path, "version")
	for k, v := range target.Env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Seconds()

	if err != nil {
		return evidence.CheckResult{
			Name:     target.Path + "/version",
			Kind:     "binary",
			Status:   "fail",
			Duration: duration,
			Output:   string(output),
			Error:    err.Error(),
		}
	}

	return evidence.CheckResult{
		Name:     target.Path + "/version",
		Kind:     "binary",
		Status:   "pass",
		Duration: duration,
		Output:   strings.TrimSpace(string(output)),
	}
}

func auditContainer(target spec.AuditTarget) []evidence.CheckResult {
	// Placeholder for container audit.
	return []evidence.CheckResult{{
		Name:   target.Path + "/container",
		Kind:   "container",
		Status: "skip",
		Error:  "container audit not yet implemented",
	}}
}

// auditEndpoint runs the contracts the target selects against its base URL,
// so the scenarios that gate a change double as a post-deploy smoke test.
// Results are named by the target path as written, so an expanded URL never
// reaches the evidence.
func auditEndpoint(target spec.AuditTarget, contracts []spec.Contract) ([]evidence.CheckResult, []evidence.ContractResult) {
	fail := func(err error) ([]evidence.CheckResult, []evidence.ContractResult) {
		return []evidence.CheckResult{{Name: target.Path + "/endpoint", Kind: "endpoint", Status: "error", Error: err.Error()}}, nil
	}

	baseURL, err := expandEnv(target.Path)
	if err != nil {
		return fail(fmt.Errorf("path: %w", err))
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return fail(fmt.Errorf("path must be an http:// or https:// base URL"))
	}

	selected, err := runner.SelectContracts(contracts, "audit target "+target.Path, target.Checks)
	if err != nil {
		return fail(err)
	}
	if len(selected) == 0 {
		return fail(fmt.Errorf("no contracts to run; add contracts to the spec or select them in checks"))
	}

	results := runner.RunContractsWith(selected, runner.ContractOptions{BaseURL: baseURL})
	for i := range results {
		results[i].Name = target.Path + "/" + results[i].Name
	}
	return nil, results
}

// expandEnv substitutes {{.env.NAME}} from the whole environment. The audit
// target is the operator's configuration, unlike contracts, which are allow-listed.
func expandEnv(s string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	tmpl, err := template.New("").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"env": env}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	numStr := s[:len(s)-1]
	unit := s[len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 's':
		multiplier = time.Second
	case 'm':
		multiplier = time.Minute
	case 'h':
		multiplier = time.Hour
	default:
		return 0, fmt.Errorf("invalid duration unit: %c", unit)
	}

	var num int
	if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", numStr)
	}

	return time.Duration(num) * multiplier, nil
}
