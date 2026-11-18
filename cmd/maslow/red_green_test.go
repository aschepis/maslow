package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
)

const grepOverlay = `contracts:
  - name: acceptance
    scenarios:
      - name: fixed
        steps:
          - action: cli
            command: grep
            args: [-q, %s, feature.txt]
          - action: assert
            expect:
              exit_code: 0
`

func TestVerify_RedGreenUsageErrors(t *testing.T) {
	cases := map[string][]string{
		"no base":    {"--red-green", "--overlay", "o.yaml"},
		"no overlay": {"--red-green", "--base", "main"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			if code := cmdVerify(args); code != 2 {
				t.Errorf("expected exit 2, got %d", code)
			}
		})
	}
}

func redGreenVerify(t *testing.T, word string) (int, *evidence.Report) {
	t.Helper()
	repoWithBranch(t,
		map[string]string{"maslow.yaml": passingSpec, "feature.txt": "old\n"},
		map[string]string{"feature.txt": "new\n"})
	tmp := t.TempDir()
	overlayPath := filepath.Join(tmp, "acceptance.yaml")
	if err := os.WriteFile(overlayPath, []byte(fmt.Sprintf(grepOverlay, word)), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(tmp, "verify.json")
	code := cmdVerify([]string{"--profile", "full", "--base", "main", "--red-green", "--overlay", overlayPath, "--report", reportPath})

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected an evidence file (exit %d): %v", code, err)
	}
	var report evidence.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return code, &report
}

func TestVerify_RedGreenPassesAContractTheChangeFixed(t *testing.T) {
	code, report := redGreenVerify(t, "new")
	if code != 0 || report.Verdict != "pass" {
		t.Errorf("expected pass with exit 0, got %s with exit %d", report.Verdict, code)
	}
}

func TestVerify_RedGreenFailsAContractAlreadyGreenOnBase(t *testing.T) {
	code, report := redGreenVerify(t, ".")
	if code != 1 || report.Verdict != "fail" {
		t.Errorf("expected fail with exit 1, got %s with exit %d", report.Verdict, code)
	}
	c := report.Contracts[0]
	if c.Status != evidence.StatusNotRed || c.Base == nil || c.Base.SHA != report.BaseSHA {
		t.Errorf("expected not_red with the base run at %s, got %+v", report.BaseSHA, c)
	}
}
