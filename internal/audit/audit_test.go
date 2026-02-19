package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRun_NoTargets(t *testing.T) {
	mas := &spec.MAS{}
	report, err := Run(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "inconclusive" {
		t.Errorf("expected inconclusive, got %s", report.Verdict)
	}
}

func TestRun_BinaryExists(t *testing.T) {
	// Create a temp "binary".
	dir := t.TempDir()
	binPath := filepath.Join(dir, "test-bin")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho test v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "binary", Path: binPath},
			},
		},
	}

	report, err := Run(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "pass" {
		t.Errorf("expected pass, got %s", report.Verdict)
	}
}

func TestRun_BinaryMissing(t *testing.T) {
	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "binary", Path: "/nonexistent/binary"},
			},
		},
	}

	report, err := Run(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected fail, got %s", report.Verdict)
	}
}

func TestRun_ContainerSkipped(t *testing.T) {
	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "container", Path: "myimage:latest"},
			},
		},
	}

	report, err := Run(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Container audit is a stub that returns skip.
	found := false
	for _, r := range report.CheckResults {
		if r.Status == "skip" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one skip result for container audit")
	}
}

func TestParseDuration(t *testing.T) {
	d, err := parseDuration("30s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Seconds() != 30 {
		t.Errorf("expected 30s, got %v", d)
	}
}
