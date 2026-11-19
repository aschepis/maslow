package evidence

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	testHead     = "1111111111111111111111111111111111111111"
	testBase     = "3333333333333333333333333333333333333333"
	otherBase    = "4444444444444444444444444444444444444444"
	testSpecHash = "aaaa"
	testOverlay  = "bbbb"
	overlayPath  = "acc.yaml"
)

func redGreenContract(name string) ContractResult {
	return ContractResult{
		Name:       name,
		Status:     "pass",
		Overlay:    overlayPath,
		HeadStatus: "pass",
		Base:       &BaseRun{SHA: testBase, Status: "fail"},
	}
}

// withRedGreen makes the report look like a passing verify --red-green run.
func withRedGreen(r *Report) {
	r.BaseSHA = testBase
	r.Overlays = []OverlayRef{{Path: overlayPath, SHA256: testOverlay}}
	r.Contracts = []ContractResult{redGreenContract("acceptance/s")}
}

func requireRedGreen(tr *Trusted) {
	tr.Base = testBase
	tr.RequireRedGreen = true
	tr.OverlaySHA256s = []string{testOverlay}
}

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
		{
			name:     "red-green satisfied",
			report:   withRedGreen,
			trusted:  requireRedGreen,
			decision: DecisionPass,
		},
		{
			name: "flag off accepts a plain overlay run",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base, r.Contracts[0].HeadStatus = nil, ""
			},
			trusted:  func(tr *Trusted) { tr.OverlaySHA256s = []string{testOverlay} },
			decision: DecisionPass,
		},
		{
			name: "plain overlay run",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base, r.Contracts[0].HeadStatus = nil, ""
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "contract base ran on another sha",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base.SHA = otherBase
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "top-level base_sha differs",
			report: func(r *Report) {
				withRedGreen(r)
				r.BaseSHA = otherBase
			},
			trusted: func(tr *Trusted) {
				tr.Base = testBase
				tr.OverlaySHA256s = []string{testOverlay}
			},
			decision: DecisionDecline,
			fields:   []string{"base_sha"},
		},
		{
			name: "base_sha missing",
			trusted: func(tr *Trusted) {
				tr.Base = testBase
			},
			decision: DecisionDecline,
			fields:   []string{"base_sha"},
		},
		{
			name: "not red",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base.Status = "pass"
				r.Contracts[0].Status = StatusNotRed
				r.Verdict = "fail"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"verdict", "contracts"},
		},
		{
			name: "base errored",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base.Status = "error"
				r.Contracts[0].Status = "error"
				r.Verdict = "fail"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"verdict", "contracts"},
		},
		{
			name: "head fails",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].HeadStatus = "fail"
				r.Contracts[0].Status = "fail"
				r.Verdict = "fail"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"verdict", "contracts"},
		},
		{
			name: "edited fields claim pass without red",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base.Status = "pass"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "one of two contracts lacks a base run",
			report: func(r *Report) {
				withRedGreen(r)
				plain := redGreenContract("acceptance/t")
				plain.Base, plain.HeadStatus = nil, ""
				r.Contracts = append(r.Contracts, plain)
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "overlay with no contract results",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts = nil
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "run used no overlays",
			report: func(r *Report) {
				r.BaseSHA = testBase
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"overlays", "contracts"},
		},
		{
			name: "spec contracts need no base run",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts = append(r.Contracts, ContractResult{Name: "spec/s", Status: "pass"})
			},
			trusted:  requireRedGreen,
			decision: DecisionPass,
		},
		{
			name: "needs review with red-green satisfied",
			report: func(r *Report) {
				withRedGreen(r)
				r.Verdict = "needs_review"
			},
			trusted:  requireRedGreen,
			decision: DecisionNeedsReview,
			fields:   []string{"verdict"},
		},
		{
			name: "pass verdict with flaky results",
			report: func(r *Report) {
				r.CheckResults = []CheckResult{{Name: "unit", Status: StatusFlaky}}
			},
			decision: DecisionPass,
		},
		{
			name: "red-green flaky on the head",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].HeadStatus = StatusFlaky
				r.Contracts[0].Status = StatusFlaky
				r.Verdict = "needs_review"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"contracts"},
		},
		{
			name: "red-green flaky on the base",
			report: func(r *Report) {
				withRedGreen(r)
				r.Contracts[0].Base.Status = StatusFlaky
				r.Contracts[0].Status = StatusNotRed
				r.Verdict = "fail"
			},
			trusted:  requireRedGreen,
			decision: DecisionDecline,
			fields:   []string{"verdict", "contracts"},
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

func TestCheck_RedGreenReasonsNameContractAndOverlay(t *testing.T) {
	r := goodReport()
	withRedGreen(r)
	r.Contracts[0].Base, r.Contracts[0].HeadStatus = nil, ""
	tr := goodTrusted()
	requireRedGreen(&tr)

	got := Check(r, tr)
	if len(got.Reasons) != 1 {
		t.Fatalf("expected one reason, got %+v", got.Reasons)
	}
	msg := got.Reasons[0].Message
	for _, want := range []string{`"acceptance/s"`, overlayPath, "--red-green"} {
		if !strings.Contains(msg, want) {
			t.Errorf("reason %q does not mention %s", msg, want)
		}
	}
}

func TestCheck_FlakyHeadIsNotGreen(t *testing.T) {
	r := goodReport()
	withRedGreen(r)
	r.Contracts[0].HeadStatus, r.Contracts[0].Status, r.Verdict = StatusFlaky, StatusFlaky, "needs_review"
	tr := goodTrusted()
	requireRedGreen(&tr)

	got := Check(r, tr)
	if len(got.Reasons) != 1 || !strings.Contains(got.Reasons[0].Message, `head status "flaky": not green`) {
		t.Errorf("expected a not-green reason for the flaky head, got %+v", got.Reasons)
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
