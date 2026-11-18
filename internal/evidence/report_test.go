package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNewReport(t *testing.T) {
	r := NewReport(Meta{Profile: "quick"})
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
	r := NewReport(Meta{Profile: "full"})
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
	r := NewReport(Meta{Profile: "full"})
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
	r := NewReport(Meta{Profile: "full"})
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
			r := NewReport(Meta{Profile: "full"})
			setup(r)
			r.ComputeVerdict()
			if r.Verdict != "fail" {
				t.Errorf("expected fail, got %s", r.Verdict)
			}
		})
		t.Run(name+" with passing checks", func(t *testing.T) {
			r := NewReport(Meta{Profile: "full"})
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
	r := NewReport(Meta{Profile: "full"})
	r.Policy = []PolicyResult{{Rule: "p", Status: "pass"}}
	r.ComputeVerdict()
	if r.Verdict != "pass" {
		t.Errorf("expected pass when policy ran and passed, got %s", r.Verdict)
	}
}

func TestComputeVerdict_OnlySkipsIsInconclusive(t *testing.T) {
	r := NewReport(Meta{Profile: "full"})
	r.CheckResults = []CheckResult{{Name: "c", Status: "skip"}}
	r.Budgets = []BudgetResult{{Name: "b", Status: "skip"}}
	r.Policy = []PolicyResult{{Rule: "g", Status: "skip"}}
	r.ComputeVerdict()
	if r.Verdict != "inconclusive" {
		t.Errorf("expected inconclusive when nothing ran, got %s", r.Verdict)
	}
}

func TestComputeVerdict_NeedsReviewPrecedence(t *testing.T) {
	review := PolicyResult{Rule: "infra/**", Kind: "gated", Status: "needs_review"}

	r := NewReport(Meta{})
	r.CheckResults = []CheckResult{{Name: "ok", Status: "pass"}}
	r.Policy = []PolicyResult{review}
	r.ComputeVerdict()
	if r.Verdict != "needs_review" {
		t.Errorf("expected needs_review when everything else is green, got %s", r.Verdict)
	}

	r = NewReport(Meta{})
	r.CheckResults = []CheckResult{{Name: "bad", Status: "fail"}}
	r.Policy = []PolicyResult{review}
	r.ComputeVerdict()
	if r.Verdict != "fail" {
		t.Errorf("expected fail to win over needs_review, got %s", r.Verdict)
	}

	r = NewReport(Meta{})
	r.Policy = []PolicyResult{review, {Rule: "x", Kind: "deny", Status: "skip"}}
	r.ComputeVerdict()
	if r.Verdict != "needs_review" {
		t.Errorf("expected needs_review to win over inconclusive, got %s", r.Verdict)
	}
}

func TestCapOutput_KeepsTailAndRecordsTruncation(t *testing.T) {
	long := strings.Repeat("noise\n", 1000) + "FAIL: the real error"
	r := NewReport(Meta{Command: "verify"})
	r.CheckResults = []CheckResult{
		{Name: "long", Status: "fail", Output: long},
		{Name: "short", Status: "pass", Output: "ok"},
	}

	r.CapOutput(100)

	got := r.CheckResults[0]
	if len(got.Output) > 100 {
		t.Errorf("expected at most 100 bytes, got %d", len(got.Output))
	}
	if !strings.HasSuffix(got.Output, "FAIL: the real error") {
		t.Errorf("expected the tail to be kept, got %q", got.Output)
	}
	if !got.OutputTruncated || got.OutputBytes != len(long) {
		t.Errorf("expected truncation recorded with %d bytes, got truncated=%v bytes=%d", len(long), got.OutputTruncated, got.OutputBytes)
	}

	short := r.CheckResults[1]
	if short.Output != "ok" || short.OutputTruncated || short.OutputBytes != 0 {
		t.Errorf("expected short output untouched, got %+v", short)
	}
}

func TestCapOutput_DoesNotSplitRunes(t *testing.T) {
	r := NewReport(Meta{})
	r.CheckResults = []CheckResult{{Output: strings.Repeat("é", 100)}}
	r.CapOutput(9)
	if !utf8.ValidString(r.CheckResults[0].Output) {
		t.Errorf("expected valid UTF-8, got %q", r.CheckResults[0].Output)
	}
}

func TestCapOutput_ZeroMeansUnlimited(t *testing.T) {
	r := NewReport(Meta{})
	r.CheckResults = []CheckResult{{Output: strings.Repeat("x", 10000)}}
	r.CapOutput(0)
	if len(r.CheckResults[0].Output) != 10000 || r.CheckResults[0].OutputTruncated {
		t.Error("expected no truncation with a zero limit")
	}
}

func TestNewReport_StampsHeader(t *testing.T) {
	r := NewReport(Meta{Command: "verify", Profile: "ci", SpecPath: "maslow.yaml", SpecSHA256: "abc", SpecSource: "base", BaseSHA: "def"})
	if r.EvidenceVersion != Version || r.MaslowVersion == "" {
		t.Errorf("expected versions stamped, got evidence=%q maslow=%q", r.EvidenceVersion, r.MaslowVersion)
	}
	if r.Command != "verify" || r.SpecSource != "base" || r.SpecSHA256 != "abc" || r.BaseSHA != "def" || r.Profile != "ci" {
		t.Errorf("expected meta copied into the report, got %+v", r)
	}
}

func TestReport_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reports", "verify.json")

	r := NewReport(Meta{Profile: "quick"})
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
