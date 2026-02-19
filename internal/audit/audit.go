// Package audit implements the maslow audit command logic.
// Audit runs in black-box mode against built artifacts or endpoints.
package audit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

const DefaultReportPath = "reports/verify.json"

// Run executes audit checks against the configured targets.
func Run(mas *spec.MAS, profile string) (*evidence.Report, error) {
	report := evidence.NewReport(profile)

	if mas.Audit == nil || len(mas.Audit.Targets) == 0 {
		report.Verdict = "inconclusive"
		return report, nil
	}

	for _, target := range mas.Audit.Targets {
		results := auditTarget(target)
		report.CheckResults = append(report.CheckResults, results...)
	}

	report.ComputeVerdict()
	return report, nil
}

func auditTarget(target spec.AuditTarget) []evidence.CheckResult {
	switch target.Kind {
	case "binary":
		return auditBinary(target)
	case "container":
		return auditContainer(target)
	case "endpoint":
		return auditEndpoint(target)
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

func auditEndpoint(target spec.AuditTarget) []evidence.CheckResult {
	// Placeholder for endpoint audit.
	return []evidence.CheckResult{{
		Name:   target.Path + "/endpoint",
		Kind:   "endpoint",
		Status: "skip",
		Error:  "endpoint audit not yet implemented",
	}}
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
