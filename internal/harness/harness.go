// Package harness manages the agentic harness lifecycle: install, update, and detach.
// It uses content generators from the scaffold package to avoid duplication.
package harness

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aschepis/maslow-agentic/internal/scaffold"
	"gopkg.in/yaml.v3"
)

// SentinelFile is the name of the file that marks a harness as detached.
const SentinelFile = ".maslow-harness-detached"

// Options configures a harness operation.
type Options struct {
	// Dir is the target project directory. Defaults to current directory.
	Dir string
	// ProjectName overrides the detected project name.
	ProjectName string
	// Force skips conflict prompts and overwrites all files.
	Force bool
	// DryRun prints what would happen without writing anything.
	DryRun bool
	// Stdin is the reader for interactive prompts. Defaults to os.Stdin.
	Stdin io.Reader
	// Stdout is the writer for output messages. Defaults to os.Stdout.
	Stdout io.Writer
}

func (o *Options) stdin() io.Reader {
	if o.Stdin != nil {
		return o.Stdin
	}
	return os.Stdin
}

func (o *Options) stdout() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

func (o *Options) dir() string {
	if o.Dir != "" {
		return o.Dir
	}
	return "."
}

// ConflictAction represents the user's choice when a harness file conflicts with an existing file.
type ConflictAction int

const (
	ActionAbort ConflictAction = iota
	ActionOverwrite
	ActionSkip
	ActionRenameAndReference
)

// HarnessFile describes a single file or directory managed by the harness.
type HarnessFile struct {
	// RelPath is the path relative to the project root.
	RelPath string
	// Content is the file content. Empty for directories.
	Content string
	// IsDir indicates this entry is a directory, not a file.
	IsDir bool
}

// Manifest returns the list of harness-managed files and directories for a project.
// maslow.yaml is NOT included — it is managed by `maslow init`.
func Manifest(projectName string) []HarnessFile {
	return []HarnessFile{
		// Directories first.
		{RelPath: "docs", IsDir: true},
		{RelPath: filepath.Join("docs", "adr"), IsDir: true},
		{RelPath: filepath.Join("docs", "templates"), IsDir: true},
		{RelPath: filepath.Join("docs", "tasks"), IsDir: true},
		{RelPath: "reports", IsDir: true},
		{RelPath: filepath.Join(".skills", "maslow"), IsDir: true},
		// Files.
		{RelPath: "CLAUDE.md", Content: scaffold.GenerateClaudeMD(projectName)},
		{RelPath: filepath.Join("docs", "MAP.md"), Content: scaffold.GenerateMapMD(projectName)},
		{RelPath: filepath.Join("docs", "PLAN.md"), Content: scaffold.GeneratePlanMD(projectName)},
		{RelPath: filepath.Join("docs", "tasks", "CONVENTION.md"), Content: scaffold.GenerateTaskConvention()},
		{RelPath: filepath.Join(".skills", "maslow", "SKILL.md"), Content: scaffold.GenerateSkillMD()},
		{RelPath: ".gitignore", Content: scaffold.GenerateGitignore()},
		{RelPath: filepath.Join("reports", ".gitkeep"), Content: ""},
	}
}

// DetectProjectName reads the project name from maslow.yaml in dir.
// Falls back to the directory basename if maslow.yaml is missing or unreadable.
func DetectProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "maslow.yaml"))
	if err != nil {
		return filepath.Base(dir)
	}

	var doc struct {
		Project struct {
			Name string `yaml:"name"`
		} `yaml:"project"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil || doc.Project.Name == "" {
		return filepath.Base(dir)
	}
	return doc.Project.Name
}

// IsDetached returns true if the sentinel file exists in dir.
func IsDetached(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, SentinelFile))
	return err == nil
}

// checkDetached returns an error if the harness is detached and force is not set.
func checkDetached(dir string, force bool) error {
	if IsDetached(dir) && !force {
		return fmt.Errorf("harness is detached (found %s); use --force to override", SentinelFile)
	}
	return nil
}
