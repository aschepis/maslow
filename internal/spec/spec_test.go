package spec

import (
	"path/filepath"
	"testing"
)

func TestLoad_ValidMinimal(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid", "minimal.yaml")
	mas, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mas.MASVersion != "1.0" {
		t.Errorf("expected mas version 1.0, got %s", mas.MASVersion)
	}
	if mas.Project.Name != "minimal-test" {
		t.Errorf("expected project name minimal-test, got %s", mas.Project.Name)
	}
}

func TestLoad_ValidFull(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid", "full.yaml")
	mas, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mas.MASVersion != "1.0" {
		t.Errorf("expected mas version 1.0, got %s", mas.MASVersion)
	}
	if mas.Project.Name != "full-test" {
		t.Errorf("expected project name full-test, got %s", mas.Project.Name)
	}
	if mas.Toolchain == nil {
		t.Fatal("expected toolchain to be set")
	}
	if len(mas.Toolchain.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(mas.Toolchain.Tools))
	}
	if mas.Checks == nil {
		t.Fatal("expected checks to be set")
	}
	if len(mas.Checks.Runner) != 4 {
		t.Errorf("expected 4 check runners, got %d", len(mas.Checks.Runner))
	}
	if len(mas.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(mas.Profiles))
	}
	if len(mas.Contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(mas.Contracts))
	}
	if len(mas.Budgets) != 2 {
		t.Errorf("expected 2 budgets, got %d", len(mas.Budgets))
	}
	if mas.Audit == nil {
		t.Fatal("expected audit to be set")
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
