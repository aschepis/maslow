package runner

import (
	"fmt"
	"slices"

	"github.com/aschepis/maslow/internal/spec"
)

// SelectContracts returns the contracts selected by name or tag; selectedBy
// names the spec location doing the selecting, for errors.
// A nil selector selects every contract.
func SelectContracts(contracts []spec.Contract, selectedBy string, selectors []string) ([]spec.Contract, error) {
	if selectors == nil {
		return contracts, nil
	}
	keys := make([][]string, len(contracts))
	for i, c := range contracts {
		keys[i] = append([]string{c.Name}, c.Tags...)
	}
	picked, err := pick(keys, selectedBy, "contract", selectors)
	if err != nil {
		return nil, err
	}
	var selected []spec.Contract
	for _, i := range picked {
		selected = append(selected, contracts[i])
	}
	return selected, nil
}

// SelectBudgets returns the budgets a profile selects by name or tag.
// A nil selector selects every budget.
func SelectBudgets(budgets []spec.Budget, selectedBy string, selectors []string) ([]spec.Budget, error) {
	if selectors == nil {
		return budgets, nil
	}
	keys := make([][]string, len(budgets))
	for i, b := range budgets {
		keys[i] = append([]string{b.Name}, b.Tags...)
	}
	picked, err := pick(keys, selectedBy, "budget", selectors)
	if err != nil {
		return nil, err
	}
	var selected []spec.Budget
	for _, i := range picked {
		selected = append(selected, budgets[i])
	}
	return selected, nil
}

// pick returns, in spec order, the indexes whose name or tags match a
// selector, and fails on a selector that matches nothing.
func pick(keys [][]string, selectedBy, kind string, selectors []string) ([]int, error) {
	for _, s := range selectors {
		if !slices.ContainsFunc(keys, func(k []string) bool { return slices.Contains(k, s) }) {
			return nil, fmt.Errorf("%s selects %s %q, but no %s has that name or tag", selectedBy, kind, s, kind)
		}
	}

	var picked []int
	for i, k := range keys {
		if slices.ContainsFunc(selectors, func(s string) bool { return slices.Contains(k, s) }) {
			picked = append(picked, i)
		}
	}
	return picked, nil
}
