package evidence

import (
	"strings"
	"testing"
)

func TestComputeVerdict_Flaky(t *testing.T) {
	flakyCheck := CheckResult{Name: "c", Status: StatusFlaky}
	review := PolicyResult{Rule: "infra/**", Kind: "gated", Status: "needs_review"}
	cases := []struct {
		name      string
		checks    []CheckResult
		contracts []ContractResult
		policy    []PolicyResult
		want      string
	}{
		{name: "only a flaky check", checks: []CheckResult{flakyCheck}, want: "pass"},
		{name: "only a flaky spec contract", contracts: []ContractResult{{Name: "c/s", Status: StatusFlaky}}, want: "pass"},
		{name: "flaky and fail", checks: []CheckResult{flakyCheck, {Name: "b", Status: "fail"}}, want: "fail"},
		{name: "flaky and needs review", checks: []CheckResult{flakyCheck}, policy: []PolicyResult{review}, want: "needs_review"},
		{
			name:      "flaky overlay contract",
			checks:    []CheckResult{{Name: "ok", Status: "pass"}},
			contracts: []ContractResult{{Name: "acc/s", Status: StatusFlaky, Overlay: "acc.yaml"}},
			want:      "needs_review",
		},
		{
			name:      "flaky overlay contract and fail",
			checks:    []CheckResult{{Name: "bad", Status: "fail"}},
			contracts: []ContractResult{{Name: "acc/s", Status: StatusFlaky, Overlay: "acc.yaml"}},
			want:      "fail",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := NewReport(Meta{})
			r.CheckResults, r.Contracts, r.Policy = c.checks, c.contracts, c.policy
			r.ComputeVerdict()
			if r.Verdict != c.want {
				t.Errorf("expected %s, got %s", c.want, r.Verdict)
			}
		})
	}
}

func TestCapOutput_AppliesToAttempts(t *testing.T) {
	long := strings.Repeat("x", 500) + "END"
	r := NewReport(Meta{})
	r.CheckResults = []CheckResult{{Name: "c", Status: StatusFlaky, Attempts: []CheckAttempt{{Status: "fail", Output: long}, {Status: "pass", Output: "ok"}}}}
	r.Contracts = []ContractResult{{
		Name:     "c/s",
		Status:   StatusNotRed,
		Attempts: []ContractAttempt{{Status: "fail", Output: long}, {Status: "pass"}},
		Base:     &BaseRun{Status: StatusFlaky, Attempts: []ContractAttempt{{Status: "fail", Output: long}, {Status: "pass"}}},
	}}

	r.CapOutput(10)

	for name, a := range map[string]struct {
		output    string
		bytes     int
		truncated bool
	}{
		"check":         {r.CheckResults[0].Attempts[0].Output, r.CheckResults[0].Attempts[0].OutputBytes, r.CheckResults[0].Attempts[0].OutputTruncated},
		"contract":      {r.Contracts[0].Attempts[0].Output, r.Contracts[0].Attempts[0].OutputBytes, r.Contracts[0].Attempts[0].OutputTruncated},
		"base contract": {r.Contracts[0].Base.Attempts[0].Output, r.Contracts[0].Base.Attempts[0].OutputBytes, r.Contracts[0].Base.Attempts[0].OutputTruncated},
	} {
		if !strings.HasSuffix(a.output, "END") || len(a.output) > 10 || !a.truncated || a.bytes != len(long) {
			t.Errorf("%s attempt not capped: %+v", name, a)
		}
	}
	if got := r.CheckResults[0].Attempts[1]; got.Output != "ok" || got.OutputTruncated {
		t.Errorf("expected short attempt output untouched, got %+v", got)
	}
}

func TestSummaryListsFlakyResults(t *testing.T) {
	r := &Report{
		Verdict: "needs_review",
		CheckResults: []CheckResult{{Name: "unit", Status: StatusFlaky, Attempts: []CheckAttempt{
			{Status: "fail", Error: "exit status 1"}, {Status: "pass"},
		}}},
		Contracts: []ContractResult{
			{Name: "acc/s", Status: StatusFlaky, Overlay: "acc.yaml", Attempts: []ContractAttempt{
				{Status: "fail", Error: "timed out"}, {Status: "fail", Error: "again"}, {Status: "pass"},
			}},
			{Name: "acc/t", Status: StatusNotRed, Overlay: "acc.yaml", Base: &BaseRun{Status: StatusFlaky, Attempts: []ContractAttempt{
				{Status: "fail", Error: "base hiccup"}, {Status: "pass"},
			}}},
		},
	}
	got := Summary(r)
	for _, want := range []string{
		"### Flaky results",
		"| check | unit | 2 | exit status 1 |",
		"| overlay contract | acc/s | 3 | timed out |",
		"| base run | acc/t | 2 | base hiccup |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary does not contain %q:\n%s", want, got)
		}
	}
}

func TestSummaryOmitsFlakySectionWithoutFlakes(t *testing.T) {
	r, err := ReadReport("../../testdata/evidence/pass.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := Summary(r); strings.Contains(got, "Flaky results") {
		t.Errorf("summary without flakes has a flaky section:\n%s", got)
	}
}
