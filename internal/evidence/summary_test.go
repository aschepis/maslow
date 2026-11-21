package evidence

import (
	"os"
	"strings"
	"testing"
)

func TestSummaryGolden(t *testing.T) {
	for _, name := range []string{"pass", "needs-review", "observe-only", "fail-with-observe", "flaky", "covers"} {
		t.Run(name, func(t *testing.T) {
			r, err := ReadReport("../../testdata/evidence/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile("../../testdata/evidence/" + name + ".summary.md")
			if err != nil {
				t.Fatal(err)
			}
			if got := Summary(r); got != string(want) {
				t.Errorf("summary mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

func TestSummaryOmitsObserveSectionWithoutObservedRules(t *testing.T) {
	r, err := ReadReport("../../testdata/evidence/pass.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := Summary(r); strings.Contains(got, "Observed policy rules") {
		t.Errorf("summary without observed rules has an observe section:\n%s", got)
	}
}

func TestSummaryKeepsObservedRowsOutOfEnforcedTable(t *testing.T) {
	r, err := ReadReport("../../testdata/evidence/fail-with-observe.json")
	if err != nil {
		t.Fatal(err)
	}
	enforced, _, _ := strings.Cut(Summary(r), "### Observed policy rules")
	if strings.Contains(enforced, "would_") || strings.Contains(enforced, "maslow.yaml |") {
		t.Errorf("enforced table lists an observed rule:\n%s", enforced)
	}
}

func TestSummaryEscapesCells(t *testing.T) {
	r := &Report{
		Verdict:      "fail",
		CheckResults: []CheckResult{{Name: "a|b", Kind: "command", Status: "fail", Error: "left | right\nnext line"}},
	}
	want := `| check | a\|b | fail | left \| right next line |`
	if got := Summary(r); !strings.Contains(got, want) {
		t.Errorf("summary does not contain %q:\n%s", want, got)
	}
}

func TestSummaryListsNotRedContracts(t *testing.T) {
	r := &Report{
		Verdict:   "fail",
		Contracts: []ContractResult{{Name: "new-flag", Status: StatusNotRed, Error: "already passed on base"}},
	}
	want := "| contract | new-flag | not_red | already passed on base |"
	if got := Summary(r); !strings.Contains(got, want) {
		t.Errorf("summary does not contain %q:\n%s", want, got)
	}
}

func TestSummaryNamesChecksThatInheritedEnv(t *testing.T) {
	r := &Report{Verdict: "pass", CheckResults: []CheckResult{
		{Name: "build", Status: "pass", InheritedEnv: true},
		{Name: "lint", Status: "pass", InheritedEnv: true},
	}}
	got := Summary(r)
	if !strings.Contains(got, "### Inherited environment") || !strings.Contains(got, "`build`, `lint`") {
		t.Errorf("expected the summary to name the checks that inherited the environment:\n%s", got)
	}
}

func TestSummaryNamesContractsAndAuditBinariesThatInheritedEnv(t *testing.T) {
	r := &Report{Verdict: "pass",
		CheckResults: []CheckResult{{Name: "bin/app/version", Kind: "binary", Status: "pass", InheritedEnv: true}},
		Contracts:    []ContractResult{{Name: "cli/help", Status: "pass", InheritedEnv: true}},
	}
	got := Summary(r)
	for _, want := range []string{"### Inherited environment", "`bin/app/version`", "`cli/help`", "inherit_env"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in the summary:\n%s", want, got)
		}
	}
}

func TestSummaryOmitsInheritedEnvSectionForScrubbedChecks(t *testing.T) {
	r, err := ReadReport("../../testdata/evidence/pass.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := Summary(r); strings.Contains(got, "Inherited environment") {
		t.Errorf("summary of scrubbed checks has an inherited-environment section:\n%s", got)
	}
}

func TestSummaryObservedPassesOnly(t *testing.T) {
	r := &Report{
		Verdict: "pass",
		Policy: []PolicyResult{
			{Rule: "a/**", Kind: "deny", Mode: ModeObserve, Status: "pass"},
			{Rule: "b/**", Kind: "deny", Mode: ModeObserve, Status: "pass"},
		},
	}
	got := Summary(r)
	if !strings.Contains(got, "### Observed policy rules (not enforced)") || !strings.Contains(got, "2 observed rules passed.") {
		t.Errorf("summary does not report the observed passes:\n%s", got)
	}
	if strings.Contains(got, "| Kind | Rule | Would be | Detail |") {
		t.Errorf("summary has an observe table with no would_* rows:\n%s", got)
	}
}

func TestSummaryObservedDetailFallsBackToFilesThenError(t *testing.T) {
	r := &Report{
		Verdict: "pass",
		Policy: []PolicyResult{
			{Rule: "a/**", Kind: "deny", Mode: ModeObserve, Status: "would_fail", Files: []string{"a/x.go", "a/y.go"}},
			{Rule: "b/**", Kind: "spec", Mode: ModeObserve, Status: "would_error", Error: "cannot read base"},
		},
	}
	got := Summary(r)
	for _, want := range []string{"| deny | a/** | would_fail | a/x.go |", "| spec | b/** | would_error | cannot read base |"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary does not contain %q:\n%s", want, got)
		}
	}
}
