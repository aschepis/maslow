package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewReport(t *testing.T) {
	r := NewReport("quick")
	if r.Profile != "quick" {
		t.Errorf("expected profile quick, got %s", r.Profile)
	}
	if r.Verdict != "inconclusive" {
		t.Errorf("expected verdict inconclusive, got %s", r.Verdict)
	}
	if r.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestComputeVerdict_Pass(t *testing.T) {
	r := NewReport("full")
	r.CheckResults = []CheckResult{
		{Name: "test1", Status: "pass"},
		{Name: "test2", Status: "pass"},
	}
	r.ComputeVerdict()
	if r.Verdict != "pass" {
		t.Errorf("expected pass, got %s", r.Verdict)
	}
}

func TestComputeVerdict_Fail(t *testing.T) {
	r := NewReport("full")
	r.CheckResults = []CheckResult{
		{Name: "test1", Status: "pass"},
		{Name: "test2", Status: "fail"},
	}
	r.ComputeVerdict()
	if r.Verdict != "fail" {
		t.Errorf("expected fail, got %s", r.Verdict)
	}
}

func TestComputeVerdict_Empty(t *testing.T) {
	r := NewReport("full")
	r.ComputeVerdict()
	if r.Verdict != "inconclusive" {
		t.Errorf("expected inconclusive, got %s", r.Verdict)
	}
}

func TestComputeVerdict_FailuresOutsideChecks(t *testing.T) {
	cases := map[string]func(r *Report){
		"contract fail":  func(r *Report) { r.Contracts = []ContractResult{{Name: "c", Status: "fail"}} },
		"contract error": func(r *Report) { r.Contracts = []ContractResult{{Name: "c", Status: "error"}} },
		"budget fail":    func(r *Report) { r.Budgets = []BudgetResult{{Name: "b", Status: "fail"}} },
		"budget error":   func(r *Report) { r.Budgets = []BudgetResult{{Name: "b", Status: "error"}} },
		"ref fail":       func(r *Report) { r.Refs = []RefResult{{Path: "p", Status: "fail"}} },
		"policy fail":    func(r *Report) { r.Policy = []PolicyResult{{Rule: "p", Status: "fail"}} },
	}
	for name, setup := range cases {
		t.Run(name+" without checks", func(t *testing.T) {
			r := NewReport("full")
			setup(r)
			r.ComputeVerdict()
			if r.Verdict != "fail" {
				t.Errorf("expected fail, got %s", r.Verdict)
			}
		})
		t.Run(name+" with passing checks", func(t *testing.T) {
			r := NewReport("full")
			r.CheckResults = []CheckResult{{Name: "ok", Status: "pass"}}
			setup(r)
			r.ComputeVerdict()
			if r.Verdict != "fail" {
				t.Errorf("expected fail, got %s", r.Verdict)
			}
		})
	}
}

func TestComputeVerdict_PassWithoutChecks(t *testing.T) {
	r := NewReport("full")
	r.Policy = []PolicyResult{{Rule: "p", Status: "pass"}}
	r.ComputeVerdict()
	if r.Verdict != "pass" {
		t.Errorf("expected pass when policy ran and passed, got %s", r.Verdict)
	}
}

func TestComputeVerdict_OnlySkipsIsInconclusive(t *testing.T) {
	r := NewReport("full")
	r.CheckResults = []CheckResult{{Name: "c", Status: "skip"}}
	r.Budgets = []BudgetResult{{Name: "b", Status: "skip"}}
	r.Policy = []PolicyResult{{Rule: "g", Status: "skip"}}
	r.ComputeVerdict()
	if r.Verdict != "inconclusive" {
		t.Errorf("expected inconclusive when nothing ran, got %s", r.Verdict)
	}
}

func TestReport_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reports", "verify.json")

	r := NewReport("quick")
	r.CheckResults = []CheckResult{
		{Name: "test1", Status: "pass", Duration: 1.5},
	}
	r.ComputeVerdict()

	if err := r.Write(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse report: %v", err)
	}

	if loaded.Verdict != "pass" {
		t.Errorf("expected pass, got %s", loaded.Verdict)
	}
	if len(loaded.CheckResults) != 1 {
		t.Errorf("expected 1 check result, got %d", len(loaded.CheckResults))
	}
}
