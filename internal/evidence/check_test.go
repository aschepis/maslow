package evidence

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testHead     = "1111111111111111111111111111111111111111"
	testSpecHash = "aaaa"
	testOverlay  = "bbbb"
)

func goodReport() *Report {
	return &Report{
		EvidenceVersion: Version,
		GitSHA:          testHead,
		SpecSHA256:      testSpecHash,
		Profile:         "quick",
		Verdict:         "pass",
	}
}

func goodTrusted() Trusted {
	return Trusted{Head: testHead, SpecSHA256: testSpecHash, Profile: "quick"}
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name     string
		report   func(*Report)
		trusted  func(*Trusted)
		decision string
		fields   []string
	}{
		{name: "everything matches", decision: DecisionPass},
		{
			name:     "profile not pinned",
			trusted:  func(tr *Trusted) { tr.Profile = "" },
			decision: DecisionPass,
		},
		{
			name:     "needs review verdict",
			report:   func(r *Report) { r.Verdict = "needs_review" },
			decision: DecisionNeedsReview,
			fields:   []string{"verdict"},
		},
		{
			name:     "unknown evidence version",
			report:   func(r *Report) { r.EvidenceVersion = "99" },
			decision: DecisionDecline,
			fields:   []string{"evidence_version"},
		},
		{
			name:     "git sha differs from head",
			report:   func(r *Report) { r.GitSHA = "2222222222222222222222222222222222222222" },
			decision: DecisionDecline,
			fields:   []string{"git_sha"},
		},
		{
			name:     "git sha missing",
			report:   func(r *Report) { r.GitSHA = "" },
			decision: DecisionDecline,
			fields:   []string{"git_sha"},
		},
		{
			name:     "spec hash differs",
			report:   func(r *Report) { r.SpecSHA256 = "cccc" },
			decision: DecisionDecline,
			fields:   []string{"spec_sha256"},
		},
		{
			name:     "profile differs",
			report:   func(r *Report) { r.Profile = "full" },
			decision: DecisionDecline,
			fields:   []string{"profile"},
		},
		{
			name:     "fail verdict",
			report:   func(r *Report) { r.Verdict = "fail" },
			decision: DecisionDecline,
			fields:   []string{"verdict"},
		},
		{
			name:     "inconclusive verdict",
			report:   func(r *Report) { r.Verdict = "inconclusive" },
			decision: DecisionDecline,
			fields:   []string{"verdict"},
		},
		{
			name:     "unknown verdict",
			report:   func(r *Report) { r.Verdict = "would_fail" },
			decision: DecisionDecline,
			fields:   []string{"verdict"},
		},
		{
			name:     "trusted overlay present",
			report:   func(r *Report) { r.Overlays = []OverlayRef{{Path: "o.yaml", SHA256: testOverlay}} },
			trusted:  func(tr *Trusted) { tr.OverlaySHA256s = []string{testOverlay} },
			decision: DecisionPass,
		},
		{
			name:     "untrusted overlay",
			report:   func(r *Report) { r.Overlays = []OverlayRef{{Path: "o.yaml", SHA256: "dddd"}} },
			decision: DecisionDecline,
			fields:   []string{"overlays"},
		},
		{
			name:     "trusted overlay missing from evidence",
			trusted:  func(tr *Trusted) { tr.OverlaySHA256s = []string{testOverlay} },
			decision: DecisionDecline,
			fields:   []string{"overlays"},
		},
		{
			name: "decline outranks needs review and lists every mismatch",
			report: func(r *Report) {
				r.GitSHA = "2222222222222222222222222222222222222222"
				r.Profile = "full"
				r.Verdict = "needs_review"
			},
			decision: DecisionDecline,
			fields:   []string{"git_sha", "profile"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, tr := goodReport(), goodTrusted()
			if c.report != nil {
				c.report(r)
			}
			if c.trusted != nil {
				c.trusted(&tr)
			}

			got := Check(r, tr)
			if got.Decision != c.decision {
				t.Errorf("decision = %q, want %q (reasons %v)", got.Decision, c.decision, got.Reasons)
			}
			var fields []string
			for _, reason := range got.Reasons {
				fields = append(fields, reason.Field)
			}
			if !reflect.DeepEqual(fields, c.fields) {
				t.Errorf("reason fields = %v, want %v", fields, c.fields)
			}
		})
	}
}

func TestReadReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "verify.json")
	if err := goodReport().Write(path); err != nil {
		t.Fatal(err)
	}

	r, err := ReadReport(path)
	if err != nil {
		t.Fatalf("ReadReport: %v", err)
	}
	if r.GitSHA != testHead || r.Verdict != "pass" {
		t.Errorf("read back %+v", r)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReport(bad); err == nil {
		t.Error("ReadReport accepted malformed JSON")
	}
	if _, err := ReadReport(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("ReadReport accepted a missing file")
	}
}

// A red-green run whose overlay contract was already green on the base must
// not be approved by the gate.
func TestCheck_DeclinesNotRedContract(t *testing.T) {
	r := goodReport()
	r.Contracts = []ContractResult{{
		Name:       "acceptance/s",
		Status:     StatusNotRed,
		HeadStatus: "pass",
		Base:       &BaseRun{SHA: testHead, Status: "pass"},
	}}
	r.ComputeVerdict()

	got := Check(r, goodTrusted())
	if got.Decision != DecisionDecline {
		t.Errorf("expected decline, got %+v", got)
	}
}
