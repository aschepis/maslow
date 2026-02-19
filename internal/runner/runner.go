// Package runner executes check runners and collects results.
package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// RunChecks executes the given check runners and returns results.
func RunChecks(checks []spec.CheckRunner) []evidence.CheckResult {
	results := make([]evidence.CheckResult, 0, len(checks))

	for _, check := range checks {
		result := runCheck(check)
		results = append(results, result)
	}

	return results
}

// RunProfileChecks executes only the checks matching the given profile.
func RunProfileChecks(mas *spec.MAS, profileName string) ([]evidence.CheckResult, error) {
	if mas.Checks == nil {
		return nil, nil
	}

	profile, ok := mas.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in spec", profileName)
	}

	// Build a set of check names in the profile.
	profileChecks := make(map[string]bool)
	for _, name := range profile.Checks {
		profileChecks[name] = true
	}

	// Filter runners to those in the profile.
	var selected []spec.CheckRunner
	for _, runner := range mas.Checks.Runner {
		if profileChecks[runner.Name] {
			selected = append(selected, runner)
		}
	}

	return RunChecks(selected), nil
}

func runCheck(check spec.CheckRunner) evidence.CheckResult {
	start := time.Now()

	timeout := 2 * time.Minute
	if check.Timeout != "" {
		if d, err := parseDuration(check.Timeout); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	parts := strings.Fields(check.Run)
	if len(parts) == 0 {
		return evidence.CheckResult{
			Name:     check.Name,
			Kind:     check.Kind,
			Status:   "error",
			Duration: time.Since(start).Seconds(),
			Error:    "empty run command",
		}
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Seconds()

	if err != nil {
		return evidence.CheckResult{
			Name:     check.Name,
			Kind:     check.Kind,
			Status:   "fail",
			Duration: duration,
			Output:   string(output),
			Error:    err.Error(),
		}
	}

	return evidence.CheckResult{
		Name:     check.Name,
		Kind:     check.Kind,
		Status:   "pass",
		Duration: duration,
		Output:   string(output),
	}
}

func parseDuration(s string) (time.Duration, error) {
	// Handle suffixes: s, m, h (matching CUE schema pattern).
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
