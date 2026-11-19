package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func fileHash(t *testing.T, path string) string {
	t.Helper()
	h, err := evidence.FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
