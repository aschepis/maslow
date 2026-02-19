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
