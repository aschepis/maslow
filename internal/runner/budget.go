package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// RunBudgets evaluates all budget constraints and returns results.
func RunBudgets(budgets []spec.Budget) []evidence.BudgetResult {
	var results []evidence.BudgetResult
	for _, budget := range budgets {
		result := runBudget(budget)
		results = append(results, result)
	}
	return results
}

func runBudget(budget spec.Budget) evidence.BudgetResult {
	switch budget.Kind {
	case "artifact_size":
		return checkArtifactSize(budget)
	case "performance":
		// Performance budgets require a benchmarking harness.
		// For v1, we validate the budget is defined but skip measurement.
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "skip",
			Error:  "performance budget measurement not yet implemented",
		}
	case "complexity":
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "skip",
			Error:  "complexity budget measurement not yet implemented",
		}
	case "availability", "rate_limit":
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("%s budget measurement not yet implemented", budget.Kind),
		}
	default:
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "error",
			Error:  fmt.Sprintf("unknown budget kind: %s", budget.Kind),
		}
	}
}

func checkArtifactSize(budget spec.Budget) evidence.BudgetResult {
	info, err := os.Stat(budget.Target)
	if err != nil {
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "fail",
			Error:  fmt.Sprintf("artifact not found: %s", budget.Target),
		}
	}

	actualBytes := info.Size()
	actualStr := formatBytes(actualBytes)

	if budget.Max == "" {
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "pass",
			Actual: actualStr,
		}
	}

	maxBytes, err := parseSize(budget.Max)
	if err != nil {
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "error",
			Error:  fmt.Sprintf("invalid max size %q: %v", budget.Max, err),
		}
	}

	if actualBytes > maxBytes {
		return evidence.BudgetResult{
			Name:   budget.Name,
			Kind:   budget.Kind,
			Status: "fail",
			Actual: actualStr,
			Limit:  budget.Max,
			Error:  fmt.Sprintf("artifact %s is %s, exceeds max %s", budget.Target, actualStr, budget.Max),
		}
	}

	return evidence.BudgetResult{
		Name:   budget.Name,
		Kind:   budget.Kind,
		Status: "pass",
		Actual: actualStr,
		Limit:  budget.Max,
	}
}

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	// Check longest suffixes first to avoid "B" matching before "MB".
	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, entry := range suffixes {
		if strings.HasSuffix(s, entry.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, entry.suffix))
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", numStr)
			}
			return int64(num * float64(entry.mult)), nil
		}
	}

	// Try plain number as bytes.
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse size: %s", s)
	}
	return num, nil
}

func formatBytes(b int64) string {
	const mb = 1024 * 1024
	if b >= mb {
		return fmt.Sprintf("%.1fMB", float64(b)/float64(mb))
	}
	const kb = 1024
	if b >= kb {
		return fmt.Sprintf("%.1fKB", float64(b)/float64(kb))
	}
	return fmt.Sprintf("%dB", b)
}
