package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Manifest & Detection ---

func TestManifest_ContainsExpectedFiles(t *testing.T) {
	m := Manifest("test-project")
	paths := make(map[string]bool)
	for _, hf := range m {
		paths[hf.RelPath] = true
	}

	expected := []string{
		"CLAUDE.md",
		filepath.Join("docs", "MAP.md"),
		filepath.Join("docs", "PLAN.md"),
		filepath.Join("docs", "tasks", "CONVENTION.md"),
		filepath.Join(".skills", "maslow", "SKILL.md"),
		".gitignore",
		filepath.Join("reports", ".gitkeep"),
	}
	for _, p := range expected {
		if !paths[p] {
			t.Errorf("manifest missing expected file: %s", p)
		}
	}
}

func TestManifest_DoesNotContainMaslowYAML(t *testing.T) {
	m := Manifest("test-project")
	for _, hf := range m {
		if hf.RelPath == "maslow.yaml" {
			t.Error("manifest should not contain maslow.yaml")
		}
	}
}

func TestDetectProjectName_FromMaslowYAML(t *testing.T) {
	dir := t.TempDir()
	content := `mas: "1.0"
project:
  name: my-cool-project
  description: "test"
`
	os.WriteFile(filepath.Join(dir, "maslow.yaml"), []byte(content), 0o644)

	name := DetectProjectName(dir)
	if name != "my-cool-project" {
		t.Errorf("expected 'my-cool-project', got %q", name)
	}
}

func TestDetectProjectName_FallbackToDirBasename(t *testing.T) {
	dir := t.TempDir()
	// No maslow.yaml — falls back to dir basename.
	name := DetectProjectName(dir)
	if name != filepath.Base(dir) {
		t.Errorf("expected %q, got %q", filepath.Base(dir), name)
	}
}

func TestIsDetached(t *testing.T) {
	dir := t.TempDir()

	if IsDetached(dir) {
		t.Error("should not be detached in empty dir")
	}

	os.WriteFile(filepath.Join(dir, SentinelFile), []byte("detached"), 0o644)

	if !IsDetached(dir) {
		t.Error("should be detached after sentinel file is created")
	}
}

// --- Install ---

func TestInstall_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check key files exist.
	for _, f := range []string{"CLAUDE.md", ".gitignore", filepath.Join("docs", "MAP.md")} {
		if _, err := os.Stat(filepath.Join(dir, f)); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", f)
		}
	}

	if !strings.Contains(out.String(), "created CLAUDE.md") {
		t.Error("output should mention created CLAUDE.md")
	}
}

func TestInstall_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// Pre-create a file.
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("old content"), 0o644)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Force:       true,
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) == "old content" {
		t.Error("CLAUDE.md should have been overwritten")
	}

	if !strings.Contains(out.String(), "overwrote CLAUDE.md (--force)") {
		t.Error("output should mention force overwrite")
	}
}

func TestInstall_PromptSkip(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// Pre-create all files that will conflict so each gets prompted.
	// We'll only test with CLAUDE.md for simplicity: send "s" for skip on every prompt.
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("existing"), 0o644)

	// Send "s\n" for each potential conflict.
	input := strings.Repeat("s\n", 20) // More than enough for all files.

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(input),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CLAUDE.md should still have original content.
	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) != "existing" {
		t.Error("CLAUDE.md should not have been modified (skipped)")
	}

	if !strings.Contains(out.String(), "skipped CLAUDE.md") {
		t.Error("output should mention skipped CLAUDE.md")
	}
}

func TestInstall_PromptOverwrite(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("old"), 0o644)

	input := strings.Repeat("o\n", 20)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(input),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) == "old" {
		t.Error("CLAUDE.md should have been overwritten")
	}
}

func TestInstall_PromptAbort(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("existing"), 0o644)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader("a\n"),
	})
	if err == nil {
		t.Fatal("expected abort error")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected abort error, got: %v", err)
	}
}

func TestInstall_PromptRenameAndReference(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("my custom guide"), 0o644)

	input := strings.Repeat("r\n", 20)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(input),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original should be renamed.
	origData, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md.original"))
	if err != nil {
		t.Fatalf("expected CLAUDE.md.original to exist: %v", err)
	}
	if string(origData) != "my custom guide" {
		t.Error("CLAUDE.md.original should contain original content")
	}

	// New file should have reference note.
	newData, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(string(newData), "CLAUDE.md.original") {
		t.Error("new CLAUDE.md should reference the original")
	}
}

func TestInstall_DryRun(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		DryRun:      true,
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No files should have been created.
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Error("CLAUDE.md should not exist in dry-run mode")
	}

	if !strings.Contains(out.String(), "would create") {
		t.Error("dry-run output should mention 'would create'")
	}
}

func TestInstall_DetachedError(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, SentinelFile), []byte("detached"), 0o644)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err == nil {
		t.Fatal("expected error for detached harness")
	}
	if !strings.Contains(err.Error(), "detached") {
		t.Errorf("expected detached error, got: %v", err)
	}
}

func TestInstall_DetachedWithForce(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, SentinelFile), []byte("detached"), 0o644)

	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Force:       true,
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}
}

func TestInstall_EOFAbortsPrompt(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("existing"), 0o644)

	// Empty reader simulates EOF.
	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err == nil {
		t.Fatal("expected abort error on EOF")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected abort error, got: %v", err)
	}
}

// --- Update ---

func TestUpdate_NoChanges(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// First install.
	err := Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Update should be a no-op.
	out.Reset()
	err = Update(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Output should not mention any updates.
	if strings.Contains(out.String(), "updated") || strings.Contains(out.String(), "created") {
		t.Errorf("update should be a no-op, but output was: %s", out.String())
	}
}

func TestUpdate_ModifiedFile(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// First install.
	Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})

	// Modify a file.
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("modified"), 0o644)

	// Update with force.
	out.Reset()
	err := Update(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Force:       true,
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) == "modified" {
		t.Error("CLAUDE.md should have been updated")
	}

	if !strings.Contains(out.String(), "updated CLAUDE.md") {
		t.Error("output should mention updated CLAUDE.md")
	}
}

func TestUpdate_DetachedError(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	os.WriteFile(filepath.Join(dir, SentinelFile), []byte("detached"), 0o644)

	err := Update(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err == nil {
		t.Fatal("expected error for detached harness")
	}
	if !strings.Contains(err.Error(), "detached") {
		t.Errorf("expected detached error, got: %v", err)
	}
}

func TestUpdate_DryRun(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// First install.
	Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})

	// Modify a file.
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("modified"), 0o644)

	// Update with dry-run.
	out.Reset()
	err := Update(Options{
		Dir:         dir,
		ProjectName: "test-project",
		DryRun:      true,
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// File should still be modified.
	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) != "modified" {
		t.Error("CLAUDE.md should not be changed in dry-run")
	}

	if !strings.Contains(out.String(), "would update") {
		t.Error("dry-run output should mention 'would update'")
	}
}

func TestUpdate_CreatesMissingFiles(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	// Install, then delete a file.
	Install(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})

	os.Remove(filepath.Join(dir, "CLAUDE.md"))

	// Update should recreate the missing file.
	out.Reset()
	err := Update(Options{
		Dir:         dir,
		ProjectName: "test-project",
		Stdout:      &out,
		Stdin:       strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); os.IsNotExist(err) {
		t.Error("CLAUDE.md should have been recreated")
	}

	if !strings.Contains(out.String(), "created CLAUDE.md") {
		t.Error("output should mention created CLAUDE.md")
	}
}

// --- Detach ---

func TestDetach_CreatesSentinel(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	err := Detach(Options{
		Dir:    dir,
		Stdout: &out,
		Stdin:  strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !IsDetached(dir) {
		t.Error("should be detached after Detach()")
	}

	data, _ := os.ReadFile(filepath.Join(dir, SentinelFile))
	if !strings.Contains(string(data), "Maslow Harness Detached") {
		t.Error("sentinel file should contain documentation")
	}
}

func TestDetach_Idempotent(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	Detach(Options{Dir: dir, Stdout: &out, Stdin: strings.NewReader("")})
	out.Reset()

	err := Detach(Options{Dir: dir, Stdout: &out, Stdin: strings.NewReader("")})
	if err != nil {
		t.Fatalf("second detach should succeed: %v", err)
	}

	if !strings.Contains(out.String(), "already detached") {
		t.Error("output should mention already detached")
	}
}

func TestDetach_DryRun(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder

	err := Detach(Options{
		Dir:    dir,
		DryRun: true,
		Stdout: &out,
		Stdin:  strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if IsDetached(dir) {
		t.Error("should not be detached in dry-run mode")
	}
}

// --- Prompt ---

func TestPromptConflict_ValidInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected ConflictAction
	}{
		{"a\n", ActionAbort},
		{"abort\n", ActionAbort},
		{"o\n", ActionOverwrite},
		{"overwrite\n", ActionOverwrite},
		{"s\n", ActionSkip},
		{"skip\n", ActionSkip},
		{"r\n", ActionRenameAndReference},
		{"rename\n", ActionRenameAndReference},
		{"rename-and-reference\n", ActionRenameAndReference},
	}

	for _, tt := range tests {
		var out strings.Builder
		action, err := promptConflict(strings.NewReader(tt.input), &out, "test.md")
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
		}
		if action != tt.expected {
			t.Errorf("input %q: expected action %d, got %d", tt.input, tt.expected, action)
		}
	}
}

func TestPromptConflict_InvalidThenValid(t *testing.T) {
	var out strings.Builder
	action, err := promptConflict(strings.NewReader("x\no\n"), &out, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionOverwrite {
		t.Errorf("expected ActionOverwrite, got %d", action)
	}
	if !strings.Contains(out.String(), "Invalid choice") {
		t.Error("should print invalid choice message")
	}
}

func TestPromptConflict_EOF(t *testing.T) {
	var out strings.Builder
	action, _ := promptConflict(strings.NewReader(""), &out, "test.md")
	if action != ActionAbort {
		t.Errorf("EOF should result in abort, got %d", action)
	}
}
