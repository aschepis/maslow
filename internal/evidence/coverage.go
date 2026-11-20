package evidence

import (
	"fmt"
	"slices"
	"sort"
)

// coveringResult is one check or contract result that names a requirement.
type coveringResult struct {
	kind    string // check, contract
	name    string
	overlay string
	status  string
	// verdictStatus is how the result counts toward the verdict, where a
	// flaky overlay contract needs review.
	verdictStatus string
}

func (c coveringResult) label() string {
	return c.kind + ":" + c.name
}

func (c coveringResult) describe() string {
	if c.overlay != "" {
		return fmt.Sprintf("%s %q (overlay %s)", c.kind, c.name, c.overlay)
	}
	return fmt.Sprintf("%s %q", c.kind, c.name)
}

// coveringResults maps each requirement ID to the results whose covers name it.
func coveringResults(r *Report) map[string][]coveringResult {
	byID := map[string][]coveringResult{}
	for _, c := range r.CheckResults {
		for _, id := range c.Covers {
			byID[id] = append(byID[id], coveringResult{kind: "check", name: c.Name, status: c.Status, verdictStatus: c.Status})
		}
	}
	for _, c := range r.Contracts {
		for _, id := range c.Covers {
			byID[id] = append(byID[id], coveringResult{
				kind: "contract", name: c.Name, overlay: c.Overlay, status: c.Status, verdictStatus: contractVerdictStatus(c),
			})
		}
	}
	return byID
}

// ComputeCoverage sets Coverage from each result's Covers, sorted by ID. It
// never changes the verdict.
func (r *Report) ComputeCoverage() {
	byID := coveringResults(r)
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	r.Coverage = nil
	for _, id := range ids {
		var labels, statuses []string
		for _, c := range byID[id] {
			labels = append(labels, c.label())
			statuses = append(statuses, c.verdictStatus)
		}
		sort.Strings(labels)
		r.Coverage = append(r.Coverage, CoverageEntry{ID: id, Status: coverageStatus(statuses), Results: slices.Compact(labels)})
	}
}

// coverageStatus applies the verdict's precedence to one requirement's results,
// with "skip" when none of them ran.
func coverageStatus(statuses []string) string {
	review, ran := false, false
	for _, s := range statuses {
		switch s {
		case "fail", "error", StatusNotRed:
			return "fail"
		case "needs_review":
			review = true
		case "pass", StatusFlaky:
			ran = true
		}
	}
	switch {
	case review:
		return "needs_review"
	case ran:
		return "pass"
	default:
		return "skip"
	}
}

// coverageMismatches returns one message per required ID that the per-result
// covers do not show as covered by results that passed.
func coverageMismatches(r *Report, ids []string, requireRedGreen bool) []string {
	byID := coveringResults(r)
	var msgs []string
	for _, id := range ids {
		if msg := coverageProblem(id, byID[id], requireRedGreen); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// Under red-green only overlay contracts count, since a spec result that also
// passed on the base proves nothing about the change.
func coverageProblem(id string, results []coveringResult, requireRedGreen bool) string {
	if len(results) == 0 {
		return fmt.Sprintf("requirement %q is not covered by any check or contract in the run", id)
	}
	if requireRedGreen {
		results = slices.DeleteFunc(slices.Clone(results), func(c coveringResult) bool { return c.overlay == "" })
		if len(results) == 0 {
			return fmt.Sprintf("requirement %q is covered by no overlay contract; --require-red-green needs one", id)
		}
	}
	for _, c := range results {
		if !coversCleanly(c.status, requireRedGreen) {
			return fmt.Sprintf("requirement %q is covered by %s with status %q", id, c.describe(), c.status)
		}
	}
	return ""
}

// A flaky overlay contract was not reliably green, so red-green accepts only pass.
func coversCleanly(status string, requireRedGreen bool) bool {
	return status == "pass" || (status == StatusFlaky && !requireRedGreen)
}
