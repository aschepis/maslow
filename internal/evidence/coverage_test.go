package evidence

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestComputeCoverage(t *testing.T) {
	cases := []struct {
		name      string
		checks    []CheckResult
		contracts []ContractResult
		want      []CoverageEntry
	}{
		{
			name:   "no covers gives no coverage",
			checks: []CheckResult{{Name: "unit", Status: "pass"}},
		},
		{
			name:      "check and contract both pass",
			checks:    []CheckResult{{Name: "unit", Status: "pass", Covers: []string{"X"}}},
			contracts: []ContractResult{{Name: "c/s", Status: "pass", Covers: []string{"X"}}},
			want:      []CoverageEntry{{ID: "X", Status: "pass", Results: []string{"check:unit", "contract:c/s"}}},
		},
		{
			name: "a pass and a fail give fail",
			contracts: []ContractResult{
				{Name: "c/a", Status: "pass", Covers: []string{"X"}},
				{Name: "c/b", Status: "fail", Covers: []string{"X"}},
			},
			want: []CoverageEntry{{ID: "X", Status: "fail", Results: []string{"contract:c/a", "contract:c/b"}}},
		},
		{
			name:      "not_red gives fail",
			contracts: []ContractResult{{Name: "c/s", Status: StatusNotRed, Overlay: "o.yaml", Covers: []string{"X"}}},
			want:      []CoverageEntry{{ID: "X", Status: "fail", Results: []string{"contract:c/s"}}},
		},
		{
			name:   "error gives fail",
			checks: []CheckResult{{Name: "unit", Status: "error", Covers: []string{"X"}}},
			want:   []CoverageEntry{{ID: "X", Status: "fail", Results: []string{"check:unit"}}},
		},
		{
			name:      "flaky overlay contract needs review",
			contracts: []ContractResult{{Name: "c/s", Status: StatusFlaky, Overlay: "o.yaml", Covers: []string{"X"}}},
			want:      []CoverageEntry{{ID: "X", Status: "needs_review", Results: []string{"contract:c/s"}}},
		},
		{
			name:      "flaky spec contract passes",
			contracts: []ContractResult{{Name: "c/s", Status: StatusFlaky, Covers: []string{"X"}}},
			want:      []CoverageEntry{{ID: "X", Status: "pass", Results: []string{"contract:c/s"}}},
		},
		{
			name:   "skip only gives skip",
			checks: []CheckResult{{Name: "unit", Status: "skip", Covers: []string{"X"}}},
			want:   []CoverageEntry{{ID: "X", Status: "skip", Results: []string{"check:unit"}}},
		},
		{
			name:      "same name for a check and a contract stays apart, sorted by id",
			checks:    []CheckResult{{Name: "same", Status: "pass", Covers: []string{"b"}}},
			contracts: []ContractResult{{Name: "same", Status: "pass", Covers: []string{"a", "b"}}},
			want: []CoverageEntry{
				{ID: "a", Status: "pass", Results: []string{"contract:same"}},
				{ID: "b", Status: "pass", Results: []string{"check:same", "contract:same"}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &Report{CheckResults: c.checks, Contracts: c.contracts}
			r.ComputeCoverage()
			if !reflect.DeepEqual(r.Coverage, c.want) {
				t.Errorf("coverage = %+v, want %+v", r.Coverage, c.want)
			}
		})
	}
}

func TestComputeCoverage_NeverChangesVerdict(t *testing.T) {
	r := &Report{CheckResults: []CheckResult{{Name: "unit", Status: "pass", Covers: []string{"X"}}}}
	r.ComputeVerdict()
	r.ComputeCoverage()
	if r.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", r.Verdict)
	}
}

func TestComputeCoverage_OmittedFromJSONWhenEmpty(t *testing.T) {
	r := &Report{CheckResults: []CheckResult{{Name: "unit", Status: "pass"}}}
	r.ComputeCoverage()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "coverage") || strings.Contains(string(data), "covers") {
		t.Errorf("expected no coverage or covers keys, got %s", data)
	}
}

func TestComputeCoverage_IsDeterministic(t *testing.T) {
	build := func() string {
		r := &Report{
			CheckResults: []CheckResult{{Name: "unit", Status: "pass", Covers: []string{"z", "a", "m"}}},
			Contracts:    []ContractResult{{Name: "c/s", Status: "pass", Covers: []string{"m", "b"}}},
		}
		r.ComputeCoverage()
		data, _ := json.Marshal(r.Coverage)
		return string(data)
	}
	first := build()
	for range 20 {
		if got := build(); got != first {
			t.Fatalf("coverage changed between runs:\n%s\n%s", first, got)
		}
	}
}
