package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/schema"
)

func TestRun_BasicScaffold(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "my-project")

	err := Run(Options{
		ProjectName: "my-project",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all expected files and directories exist.
	expectedFiles := []string{
		"maslow.yaml",
		"CLAUDE.md",
		".gitignore",
		filepath.Join("docs", "MAP.md"),
		filepath.Join("docs", "PLAN.md"),
		filepath.Join("docs", "tasks", "CONVENTION.md"),
		filepath.Join("reports", ".gitkeep"),
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	expectedDirs := []string{
		"docs",
		filepath.Join("docs", "adr"),
		filepath.Join("docs", "templates"),
		filepath.Join("docs", "tasks"),
		"reports",
	}
	for _, d := range expectedDirs {
		path := filepath.Join(projectDir, d)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", d)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestRun_DefaultDir(t *testing.T) {
	// When Dir is empty, it defaults to ProjectName.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := Run(Options{ProjectName: "test-proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "test-proj", "maslow.yaml")); os.IsNotExist(err) {
		t.Error("expected maslow.yaml to exist in test-proj/")
	}
}

func TestRun_MaslowYAMLContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "my-app")

	err := Run(Options{
		ProjectName: "my-app",
		Dir:         projectDir,
		Description: "My awesome app",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "maslow.yaml"))
	if err != nil {
		t.Fatalf("failed to read maslow.yaml: %v", err)
	}
	content := string(data)

	checks := []string{
		`mas: "1.0"`,
		`name: my-app`,
		`description: "My awesome app"`,
		`name: build`,
		`name: test`,
		`name: lint`,
		`name: self-validate`,
		`kind: command`,
		`profiles:`,
		`quick:`,
		`full:`,
		`policy:`,
		`deny:`,
		`protected:`,
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("maslow.yaml missing expected content: %q", check)
		}
	}
}

func TestRun_MaslowYAMLWithToolchain(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "tc-project")

	err := Run(Options{
		ProjectName: "tc-project",
		Dir:         projectDir,
		Toolchain:   "mise",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "maslow.yaml"))
	if err != nil {
		t.Fatalf("failed to read maslow.yaml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "manager: mise") {
		t.Error("maslow.yaml missing toolchain manager")
	}
}

func TestRun_MaslowYAMLWithoutToolchain(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "no-tc")

	err := Run(Options{
		ProjectName: "no-tc",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "maslow.yaml"))
	if err != nil {
		t.Fatalf("failed to read maslow.yaml: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "toolchain:") {
		t.Error("maslow.yaml should not contain toolchain section when no toolchain specified")
	}
}

func TestRun_CLAUDEMDContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "agent-project")

	err := Run(Options{
		ProjectName: "agent-project",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	content := string(data)

	// Check that the agentic harness principles are present.
	expectedSections := []string{
		"Agent Guide",
		"Operating Principles",
		"Non-Negotiable Behaviors",
		"Task System",
		"Process for New Work",
		"Key Paths",
		"Conventions",
		"Verification",
		"Adding a New Feature",
		"Harness Propagation Rule",
		"Repository knowledge is the system of record",
		"Humans steer, agents execute",
		"Agent legibility is the goal",
		"Encode golden principles into the repo",
		"Small increments",
		"Parallelize wherever possible",
		"maslow verify --profile quick",
		"docs/adr/",
		"docs/templates/",
		"docs/tasks/",
		"docs/tasks/CONVENTION.md",
		"Architecture Decision Records",
		"Scan frontmatter only",
		"status: todo",
		"assigned_to",
	}
	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("CLAUDE.md missing expected content: %q", section)
		}
	}

	// Verify project name appears in the header.
	if !strings.Contains(content, "# agent-project") {
		t.Error("CLAUDE.md should contain project name in header")
	}
}

func TestRun_MapMDContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "map-test")

	err := Run(Options{
		ProjectName: "map-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "docs", "MAP.md"))
	if err != nil {
		t.Fatalf("failed to read docs/MAP.md: %v", err)
	}
	content := string(data)

	expectedContent := []string{
		"Repository Map",
		"Architecture Overview",
		"Key Entrypoints",
		"Canonical File Locations",
		"Multi-Agent Conventions",
		"maslow verify",
		"docs/tasks/",
		"Task convention",
	}
	for _, check := range expectedContent {
		if !strings.Contains(content, check) {
			t.Errorf("docs/MAP.md missing expected content: %q", check)
		}
	}
}

func TestRun_PlanMDContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "plan-test")

	err := Run(Options{
		ProjectName: "plan-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "docs", "PLAN.md"))
	if err != nil {
		t.Fatalf("failed to read docs/PLAN.md: %v", err)
	}
	content := string(data)

	expectedContent := []string{
		"Execution Plan",
		"Milestones",
		"Parallel Workstreams",
		"Risk Register",
		"Definition of Done",
		"maslow verify",
	}
	for _, check := range expectedContent {
		if !strings.Contains(content, check) {
			t.Errorf("docs/PLAN.md missing expected content: %q", check)
		}
	}
}

func TestRun_GitignoreContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "gi-test")

	err := Run(Options{
		ProjectName: "gi-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "reports/") {
		t.Error(".gitignore should exclude reports/")
	}
	if !strings.Contains(content, ".env") {
		t.Error(".gitignore should exclude .env")
	}
}

func TestRun_DefaultDescription(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "desc-test")

	err := Run(Options{
		ProjectName: "desc-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "maslow.yaml"))
	if err != nil {
		t.Fatalf("failed to read maslow.yaml: %v", err)
	}

	if !strings.Contains(string(data), "TODO: describe your project") {
		t.Error("expected default description placeholder")
	}
}

func TestRun_EmptyProjectName(t *testing.T) {
	err := Run(Options{})
	if err == nil {
		t.Fatal("expected error for empty project name")
	}
	if !strings.Contains(err.Error(), "project name is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRun_MaslowYAMLIsSchemaValid(t *testing.T) {
	// This test verifies the generated maslow.yaml validates against the CUE schema
	// by running it through the actual schema validation pipeline.
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "valid-test")

	err := Run(Options{
		ProjectName: "valid-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(projectDir, "maslow.yaml")

	result, err := schema.ValidateFile(specPath)
	if err != nil {
		t.Fatalf("schema validation error: %v", err)
	}
	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("schema validation error: %s", e.Message)
		}
		t.Fatal("generated maslow.yaml is not valid against CUE schema")
	}
}

func TestRun_MaslowYAMLWithToolchainIsSchemaValid(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "tc-valid")

	err := Run(Options{
		ProjectName: "tc-valid",
		Dir:         projectDir,
		Toolchain:   "asdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(projectDir, "maslow.yaml")

	result, err := schema.ValidateFile(specPath)
	if err != nil {
		t.Fatalf("schema validation error: %v", err)
	}
	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("schema validation error: %s", e.Message)
		}
		t.Fatal("generated maslow.yaml with toolchain is not valid against CUE schema")
	}
}

func TestRun_TaskConventionContent(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "task-test")

	err := Run(Options{
		ProjectName: "task-test",
		Dir:         projectDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "docs", "tasks", "CONVENTION.md"))
	if err != nil {
		t.Fatalf("failed to read docs/tasks/CONVENTION.md: %v", err)
	}
	content := string(data)

	expectedContent := []string{
		"Task Convention",
		"Status Lifecycle",
		"Agent Protocol",
		"Discovering Tasks",
		"Claiming a Task",
		"File Format",
		"Frontmatter Fields",
		"Naming Convention",
		"draft",
		"todo",
		"in_progress",
		"done",
		"blocked",
		"assigned_to",
		"depends_on",
		"git pull",
	}
	for _, check := range expectedContent {
		if !strings.Contains(content, check) {
			t.Errorf("docs/tasks/CONVENTION.md missing expected content: %q", check)
		}
	}
}

func TestRun_IdempotentOverwrite(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "idempotent")

	// Run scaffold twice.
	opts := Options{
		ProjectName: "idempotent",
		Dir:         projectDir,
	}

	if err := Run(opts); err != nil {
		t.Fatalf("first scaffold failed: %v", err)
	}

	if err := Run(opts); err != nil {
		t.Fatalf("second scaffold failed: %v", err)
	}

	// Verify file still exists and is valid.
	if _, err := os.Stat(filepath.Join(projectDir, "maslow.yaml")); os.IsNotExist(err) {
		t.Error("maslow.yaml should exist after re-scaffold")
	}
}
