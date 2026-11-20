package runner

import (
	"fmt"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
)

// shouldRetry says whether an attempt with this status gets another try. An
// error is a configuration problem that running again cannot fix.
func shouldRetry(status string) bool {
	return status == "fail"
}

// finalStatus names a result from the statuses of its attempts, in order.
// Attempts stop at the first pass, so a pass after the first is a flake.
func finalStatus(attempts []string) string {
	last := attempts[len(attempts)-1]
	if last == "pass" && len(attempts) > 1 {
		return evidence.StatusFlaky
	}
	return last
}

// foldCheckAttempts combines a check's attempts into one result described by
// the last attempt, with the total duration and every attempt listed.
func foldCheckAttempts(attempts []evidence.CheckResult) evidence.CheckResult {
	result := attempts[len(attempts)-1]
	if len(attempts) == 1 {
		return result
	}

	var statuses []string
	result.Duration = 0
	for _, a := range attempts {
		statuses = append(statuses, a.Status)
		result.Duration += a.Duration
		result.Attempts = append(result.Attempts, evidence.CheckAttempt{
			Status:   a.Status,
			Duration: a.Duration,
			Output:   a.Output,
			Error:    a.Error,
		})
	}
	result.Status = finalStatus(statuses)
	return result
}

// foldContractAttempts combines a scenario's attempts into one result
// described by the last attempt, with every attempt listed.
func foldContractAttempts(attempts []evidence.ContractResult) evidence.ContractResult {
	result := attempts[len(attempts)-1]
	if len(attempts) == 1 {
		return result
	}

	var statuses []string
	for _, a := range attempts {
		statuses = append(statuses, a.Status)
		result.Attempts = append(result.Attempts, evidence.ContractAttempt{
			Status:       a.Status,
			Error:        a.Error,
			FailedStep:   a.FailedStep,
			Output:       a.Output,
			Artifacts:    a.Artifacts,
			Observations: a.Observations,
		})
	}
	result.Status = finalStatus(statuses)
	return result
}

// attemptScreenshotPath gives each retry its own screenshot, so a retry never
// removes the screenshot of the failure before it.
func attemptScreenshotPath(path string, attempt int) string {
	if attempt == 0 {
		return path
	}
	return fmt.Sprintf("%s.attempt-%d.png", strings.TrimSuffix(path, ".png"), attempt+1)
}
