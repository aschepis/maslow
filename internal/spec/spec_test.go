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
	// Verify MCP refs are parsed correctly.
	if len(mas.Refs) != 4 {
		t.Fatalf("expected 4 refs, got %d", len(mas.Refs))
	}
	mcpRef := mas.Refs[2]
	if mcpRef.Kind != "mcp" {
		t.Errorf("expected ref kind mcp, got %s", mcpRef.Kind)
	}
	if mcpRef.Name != "github" {
		t.Errorf("expected ref name github, got %s", mcpRef.Name)
	}
	if mcpRef.Transport != "stdio" {
		t.Errorf("expected transport stdio, got %s", mcpRef.Transport)
	}
	if mcpRef.Command != "npx" {
		t.Errorf("expected command npx, got %s", mcpRef.Command)
	}
	if len(mcpRef.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(mcpRef.Args))
	}
	if mcpRef.Env["GITHUB_TOKEN"] != "" {
		t.Errorf("expected empty GITHUB_TOKEN, got %s", mcpRef.Env["GITHUB_TOKEN"])
	}
	// Verify minimal MCP ref (no optional fields).
	minimalMCP := mas.Refs[3]
	if minimalMCP.Kind != "mcp" {
		t.Errorf("expected ref kind mcp, got %s", minimalMCP.Kind)
	}
	if minimalMCP.Name != "browser" {
		t.Errorf("expected ref name browser, got %s", minimalMCP.Name)
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
