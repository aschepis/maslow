package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
)

const (
	fixtureHead = "1111111111111111111111111111111111111111"
	fixtureSpec = "../../testdata/valid/minimal.yaml"
	passFile    = "../../testdata/evidence/pass.json"
	reviewFile  = "../../testdata/evidence/needs-review.json"

	fixtureBase      = "3333333333333333333333333333333333333333"
	fixtureOverlay   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	redGreenFile     = "../../testdata/evidence/red-green.json"
	plainOverlayFile = "../../testdata/evidence/overlay-without-red-green.json"
)

func TestCmdEvidence(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		want     int
		contains string
	}{
		{"pass", []string{"check", passFile, "--head", fixtureHead, "--spec", fixtureSpec, "--profile", "quick"}, 0, "pass"},
		{"flags before file", []string{"check", "--head", fixtureHead, "--spec", fixtureSpec, passFile}, 0, "pass"},
		{"needs review", []string{"check", reviewFile, "--head", fixtureHead, "--spec", fixtureSpec}, 1, "verdict is needs_review"},
		{"wrong head", []string{"check", passFile, "--head", "2222", "--spec", fixtureSpec}, 1, passFile + ": git_sha"},
		{"wrong spec", []string{"check", passFile, "--head", fixtureHead, "--spec", "../../testdata/valid/full.yaml"}, 1, "spec_sha256"},
		{"wrong profile", []string{"check", passFile, "--head", fixtureHead, "--spec", fixtureSpec, "--profile", "full"}, 1, "profile"},
		{"missing evidence", []string{"check", "no-such.json", "--head", fixtureHead, "--spec", fixtureSpec}, 1, "decline"},
		{"missing file argument", []string{"check", "--head", fixtureHead, "--spec", fixtureSpec}, 2, ""},
		{"two file arguments", []string{"check", passFile, reviewFile, "--head", fixtureHead, "--spec", fixtureSpec}, 2, ""},
		{"missing head", []string{"check", passFile, "--spec", fixtureSpec}, 2, ""},
		{"missing spec", []string{"check", passFile, "--head", fixtureHead}, 2, ""},
		{"unreadable trusted spec", []string{"check", passFile, "--head", fixtureHead, "--spec", "no-such.yaml"}, 2, ""},
		{"require red-green without base", []string{"check", redGreenFile, "--head", fixtureHead, "--spec", fixtureSpec, "--overlay-sha256", fixtureOverlay, "--require-red-green"}, 2, ""},
		{"require red-green without overlay", []string{"check", redGreenFile, "--head", fixtureHead, "--spec", fixtureSpec, "--base", fixtureBase, "--require-red-green"}, 2, ""},
		{"red-green evidence passes", []string{"check", redGreenFile, "--head", fixtureHead, "--spec", fixtureSpec, "--overlay-sha256", fixtureOverlay, "--base", fixtureBase, "--require-red-green"}, 0, "pass"},
		{"plain overlay evidence declines", []string{"check", plainOverlayFile, "--head", fixtureHead, "--spec", fixtureSpec, "--overlay-sha256", fixtureOverlay, "--base", fixtureBase, "--require-red-green"}, 1, "--red-green"},
		{"base alone checks base_sha", []string{"check", plainOverlayFile, "--head", fixtureHead, "--spec", fixtureSpec, "--overlay-sha256", fixtureOverlay, "--base", fixtureHead}, 1, "base_sha"},
		{"require covered empty", []string{"check", passFile, "--head", fixtureHead, "--spec", fixtureSpec, "--require-covered", ""}, 2, ""},
		{"require covered with whitespace", []string{"check", passFile, "--head", fixtureHead, "--spec", fixtureSpec, "--require-covered", "two words"}, 2, ""},
		{"require covered uncovered", []string{"check", passFile, "--head", fixtureHead, "--spec", fixtureSpec, "--require-covered", "acme/app#8"}, 1, `requirement "acme/app#8" is not covered`},
		{"summary", []string{"summary", "../../testdata/evidence/observe-only.json"}, 0, "Observed policy rules"},
		{"summary lists requirements", []string{"summary", "../../testdata/evidence/covers.json"}, 0, "### Requirements"},
		{"summary of missing evidence", []string{"summary", "no-such.json"}, 1, ""},
		{"summary without file", []string{"summary"}, 2, ""},
		{"summary with two files", []string{"summary", passFile, reviewFile}, 2, ""},
		{"no subcommand", nil, 2, ""},
		{"unknown subcommand", []string{"inspect"}, 2, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			if got := cmdEvidence(c.args, &out); got != c.want {
				t.Errorf("exit = %d, want %d; output:\n%s", got, c.want, out.String())
			}
			if !strings.Contains(out.String(), c.contains) {
				t.Errorf("output %q does not contain %q", out.String(), c.contains)
			}
		})
	}
}

func TestCmdEvidenceJSON(t *testing.T) {
	var out bytes.Buffer
	code := cmdEvidence([]string{"check", passFile, "--head", "2222", "--spec", fixtureSpec, "--json"}, &out)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	var got struct {
		Evidence string `json:"evidence"`
		Decision string `json:"decision"`
		Reasons  []struct {
			Field string `json:"field"`
		} `json:"reasons"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if got.Evidence != passFile || got.Decision != "decline" || len(got.Reasons) != 1 || got.Reasons[0].Field != "git_sha" {
		t.Errorf("unexpected JSON: %+v", got)
	}
}

func TestCmdEvidenceOverlays(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(overlay, []byte("contracts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const otherOverlayHash = "c7a0ef9ef1a3b4ed1e4a9fd6c3a4afb0f68b5f0f6be5f6fb4c9b4e3b8b4c3e2a"

	data, err := os.ReadFile(passFile)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	report["overlays"] = []map[string]string{{"path": "extra.yaml", "sha256": fileHash(t, overlay)}}
	withOverlay := filepath.Join(dir, "verify.json")
	data, _ = json.Marshal(report)
	if err := os.WriteFile(withOverlay, data, 0o644); err != nil {
		t.Fatal(err)
	}

	base := []string{"check", withOverlay, "--head", fixtureHead, "--spec", fixtureSpec}
	cases := []struct {
		name  string
		extra []string
		want  int
	}{
		{"overlay file trusted", []string{"--overlay", overlay}, 0},
		{"overlay hash trusted", []string{"--overlay-sha256", fileHash(t, overlay)}, 0},
		{"overlay not trusted", nil, 1},
		{"different overlay trusted", []string{"--overlay-sha256", otherOverlayHash}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			if got := cmdEvidence(append(append([]string{}, base...), c.extra...), &out); got != c.want {
				t.Errorf("exit = %d, want %d; output:\n%s", got, c.want, out.String())
			}
		})
	}
}

func TestCmdEvidenceRequireRedGreenEndToEnd(t *testing.T) {
	repoWithBranch(t,
		map[string]string{"maslow.yaml": passingSpec, "feature.txt": "old\n"},
		map[string]string{"feature.txt": "new\n"})
	head := gitOutput(t, "rev-parse", "HEAD")
	base := gitOutput(t, "merge-base", "main", "HEAD")
	tmp := t.TempDir()
	overlay := filepath.Join(tmp, "acceptance.yaml")
	if err := os.WriteFile(overlay, []byte(fmt.Sprintf(grepOverlay, "new")), 0o644); err != nil {
		t.Fatal(err)
	}

	check := func(report string) (int, string) {
		var out bytes.Buffer
		code := cmdEvidence([]string{"check", report, "--head", head, "--spec", "maslow.yaml",
			"--overlay", overlay, "--base", base, "--require-red-green"}, &out)
		return code, out.String()
	}

	redGreen := filepath.Join(tmp, "red-green.json")
	if code := cmdVerify([]string{"--profile", "full", "--base", "main", "--red-green", "--overlay", overlay, "--report", redGreen}); code != 0 {
		t.Fatalf("verify --red-green exit = %d", code)
	}
	if code, out := check(redGreen); code != 0 {
		t.Errorf("red-green evidence: exit = %d, want 0; output:\n%s", code, out)
	}

	plain := filepath.Join(tmp, "plain.json")
	if code := cmdVerify([]string{"--profile", "full", "--base", "main", "--overlay", overlay, "--report", plain}); code != 0 {
		t.Fatalf("verify --overlay exit = %d", code)
	}
	code, out := check(plain)
	if code != 1 || !strings.Contains(out, "--red-green") {
		t.Errorf("plain overlay evidence: exit = %d, want 1 with a --red-green reason; output:\n%s", code, out)
	}
}

func TestCmdEvidenceRequireCoveredEndToEnd(t *testing.T) {
	repoWithBranch(t,
		map[string]string{"maslow.yaml": passingSpec, "feature.txt": "old\n"},
		map[string]string{"feature.txt": "new\n"})
	head := gitOutput(t, "rev-parse", "HEAD")
	base := gitOutput(t, "merge-base", "main", "HEAD")
	tmp := t.TempDir()
	overlay := filepath.Join(tmp, "acceptance.yaml")
	covered := strings.Replace(fmt.Sprintf(grepOverlay, "new"), "  - name: acceptance\n", "  - name: acceptance\n    covers: [acme/app#7]\n", 1)
	if err := os.WriteFile(overlay, []byte(covered), 0o644); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(tmp, "verify.json")
	if code := cmdVerify([]string{"--profile", "full", "--base", "main", "--red-green", "--overlay", overlay, "--report", report}); code != 0 {
		t.Fatalf("verify --red-green exit = %d", code)
	}

	check := func(id string) (int, string) {
		var out bytes.Buffer
		code := cmdEvidence([]string{"check", report, "--head", head, "--spec", "maslow.yaml",
			"--overlay", overlay, "--base", base, "--require-red-green", "--require-covered", id}, &out)
		return code, out.String()
	}
	if code, out := check("acme/app#7"); code != 0 {
		t.Errorf("covered requirement: exit = %d, want 0; output:\n%s", code, out)
	}
	if code, out := check("acme/app#8"); code != 1 || !strings.Contains(out, `"acme/app#8"`) {
		t.Errorf("uncovered requirement: exit = %d, want 1 naming acme/app#8; output:\n%s", code, out)
	}
}

func gitOutput(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	h, err := evidence.FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
