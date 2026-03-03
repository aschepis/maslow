// Maslow CLI — executable specification system for agent-built software.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aschepis/maslow-agentic/internal/audit"
	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/harness"
	"github.com/aschepis/maslow-agentic/internal/scaffold"
	"github.com/aschepis/maslow-agentic/internal/schema"
	"github.com/aschepis/maslow-agentic/internal/spec"
	"github.com/aschepis/maslow-agentic/internal/verify"
)

var (
	// Set via ldflags at build time.
	version = "dev"
	gitSHA  = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		os.Exit(cmdValidate(os.Args[2:]))
	case "verify":
		os.Exit(cmdVerify(os.Args[2:]))
	case "audit":
		os.Exit(cmdAudit(os.Args[2:]))
	case "scaffold":
		os.Exit(cmdScaffold(os.Args[2:]))
	case "harness":
		os.Exit(cmdHarness(os.Args[2:]))
	case "init":
		os.Exit(cmdInit(os.Args[2:]))
	case "version":
		os.Exit(cmdVersion())
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "maslow: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: maslow <command> [options]

Commands:
  validate <file>              Validate a maslow.yaml against the CUE schema
  verify   --profile <name>    Run verification checks
  audit    --profile <name>    Run black-box audit
  scaffold <name> [options]    Scaffold a new Maslow-managed project with agentic harness
  harness  <subcommand>        Manage the agentic harness (install, update, detach)
  init     [--apply]           Initialize or scaffold maslow.yaml in current directory
  version                      Print version information`)
}

func cmdValidate(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "maslow validate: missing file argument")
		fmt.Fprintln(os.Stderr, "Usage: maslow validate <file>")
		return 2
	}

	path := args[0]

	result, err := schema.ValidateFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow validate: %v\n", err)
		return 1
	}

	if !result.Valid {
		fmt.Fprintf(os.Stderr, "maslow validate: %s is invalid\n", path)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e.Message)
		}
		return 1
	}

	fmt.Printf("maslow validate: %s is valid\n", path)
	return 0
}

func cmdVerify(args []string) int {
	profile := "quick"
	specPath := "maslow.yaml"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow verify: --profile requires a value")
				return 2
			}
			i++
			profile = args[i]
		case "--spec":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow verify: --spec requires a value")
				return 2
			}
			i++
			specPath = args[i]
		default:
			fmt.Fprintf(os.Stderr, "maslow verify: unknown option %q\n", args[i])
			return 2
		}
	}

	// Validate spec first.
	result, err := schema.ValidateFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		return 1
	}
	if !result.Valid {
		fmt.Fprintf(os.Stderr, "maslow verify: %s is invalid\n", specPath)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e.Message)
		}
		return 1
	}

	// Load spec.
	mas, err := spec.Load(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		return 1
	}

	// Run verification.
	report, err := verify.Run(mas, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		return 1
	}

	// Write report.
	reportPath := "reports/verify.json"
	if err := report.Write(reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		return 1
	}

	printReport(report, reportPath)

	if report.Verdict != "pass" {
		return 1
	}
	return 0
}

func cmdAudit(args []string) int {
	profile := "full"
	specPath := "maslow.yaml"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow audit: --profile requires a value")
				return 2
			}
			i++
			profile = args[i]
		case "--spec":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow audit: --spec requires a value")
				return 2
			}
			i++
			specPath = args[i]
		default:
			fmt.Fprintf(os.Stderr, "maslow audit: unknown option %q\n", args[i])
			return 2
		}
	}

	// Validate spec first.
	result, err := schema.ValidateFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}
	if !result.Valid {
		fmt.Fprintf(os.Stderr, "maslow audit: %s is invalid\n", specPath)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e.Message)
		}
		return 1
	}

	// Load spec.
	mas, err := spec.Load(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}

	// Run audit.
	report, err := audit.Run(mas, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}

	// Write report.
	reportPath := "reports/verify.json"
	if err := report.Write(reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}

	printReport(report, reportPath)

	if report.Verdict != "pass" {
		return 1
	}
	return 0
}

func cmdScaffold(args []string) int {
	var name, dir, description string

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow scaffold: --dir requires a value")
				return 2
			}
			i++
			dir = args[i]
		case "--description":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "maslow scaffold: --description requires a value")
				return 2
			}
			i++
			description = args[i]
		default:
			if name == "" && !isFlag(args[i]) {
				name = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "maslow scaffold: unknown option %q\n", args[i])
				fmt.Fprintln(os.Stderr, "Usage: maslow scaffold <name> [--dir <path>] [--description <text>]")
				return 2
			}
		}
		i++
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "maslow scaffold: missing project name")
		fmt.Fprintln(os.Stderr, "Usage: maslow scaffold <name> [--dir <path>] [--description <text>]")
		return 2
	}

	opts := scaffold.Options{
		ProjectName: name,
		Dir:         dir,
		Description: description,
	}

	// Detect toolchain in the target directory if it already exists.
	targetDir := dir
	if targetDir == "" {
		targetDir = name
	}
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		opts.Toolchain = detectToolchainManagerIn(targetDir)
		if opts.Toolchain != "" {
			fmt.Printf("maslow scaffold: detected toolchain manager: %s\n", opts.Toolchain)
		}
	}

	if err := scaffold.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "maslow scaffold: %v\n", err)
		return 1
	}

	fmt.Printf("maslow scaffold: created project %q in %s/\n", name, targetDir)
	fmt.Println("maslow scaffold: files created:")
	fmt.Println("  maslow.yaml    — project spec")
	fmt.Println("  CLAUDE.md      — agentic harness guide")
	fmt.Println("  docs/MAP.md    — architecture map")
	fmt.Println("  docs/PLAN.md   — execution plan")
	fmt.Println("  docs/adr/      — architecture decision records")
	fmt.Println("  docs/templates/— decision templates")
	fmt.Println("  docs/tasks/    — task queue for agents")
	fmt.Println("  .gitignore     — git ignore rules")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Println("  # Edit maslow.yaml to configure your checks")
	fmt.Println("  # Edit CLAUDE.md to describe your project")
	fmt.Println("  maslow validate maslow.yaml")
	return 0
}

func cmdHarness(args []string) int {
	if len(args) == 0 {
		printHarnessUsage()
		return 2
	}

	switch args[0] {
	case "install":
		return cmdHarnessInstall(args[1:])
	case "update":
		return cmdHarnessUpdate(args[1:])
	case "detach":
		return cmdHarnessDetach(args[1:])
	case "help", "--help", "-h":
		printHarnessUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "maslow harness: unknown subcommand %q\n", args[0])
		printHarnessUsage()
		return 2
	}
}

func printHarnessUsage() {
	fmt.Fprintln(os.Stderr, `Usage: maslow harness <subcommand> [options]

Subcommands:
  install    Install the agentic harness into a project
  update     Update harness files to the latest version
  detach     Detach the harness to prevent future updates

Options:
  --dir <path>    Target directory (default: current directory)
  --force         Overwrite existing files without prompting
  --dry-run       Show what would happen without writing anything`)
}

func parseHarnessFlags(args []string) harness.Options {
	var opts harness.Options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				i++
				opts.Dir = args[i]
			}
		case "--force":
			opts.Force = true
		case "--dry-run":
			opts.DryRun = true
		}
	}
	opts.Stdin = os.Stdin
	opts.Stdout = os.Stdout
	return opts
}

func cmdHarnessInstall(args []string) int {
	opts := parseHarnessFlags(args)
	if err := harness.Install(opts); err != nil {
		fmt.Fprintf(os.Stderr, "maslow: %v\n", err)
		return 1
	}
	return 0
}

func cmdHarnessUpdate(args []string) int {
	opts := parseHarnessFlags(args)
	if err := harness.Update(opts); err != nil {
		fmt.Fprintf(os.Stderr, "maslow: %v\n", err)
		return 1
	}
	return 0
}

func cmdHarnessDetach(args []string) int {
	opts := parseHarnessFlags(args)
	if err := harness.Detach(opts); err != nil {
		fmt.Fprintf(os.Stderr, "maslow: %v\n", err)
		return 1
	}
	return 0
}

func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func detectToolchainManagerIn(dir string) string {
	check := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	if check(".mise.toml") || check(".mise") {
		return "mise"
	}
	if check(".tool-versions") {
		return "asdf"
	}
	if check("flake.nix") || check("shell.nix") {
		return "nix"
	}
	return ""
}

func cmdInit(args []string) int {
	apply := false
	for _, arg := range args {
		if arg == "--apply" {
			apply = true
		}
	}

	specPath := "maslow.yaml"

	if _, err := os.Stat(specPath); err == nil {
		if !apply {
			fmt.Printf("maslow init: %s already exists (use --apply to overwrite)\n", specPath)
			return 0
		}
	}

	manager := detectToolchainManager()
	toolchainSection := ""
	if manager != "" {
		toolchainSection = fmt.Sprintf(`
toolchain:
  manager: %s
`, manager)
		fmt.Printf("maslow init: detected toolchain manager: %s\n", manager)
	}

	template := fmt.Sprintf(`mas: "1.0"
project:
  name: my-project
  description: "TODO: describe your project"
%s
checks:
  runner:
    - name: build
      kind: command
      run: "echo 'TODO: add build command'"
      timeout: 120s
      tags:
        - build
    - name: test
      kind: command
      run: "echo 'TODO: add test command'"
      timeout: 300s
      tags:
        - test

profiles:
  quick:
    description: Fast checks only
    checks:
      - build
  full:
    description: All checks
    checks:
      - build
      - test
`, toolchainSection)

	if apply || !fileExists(specPath) {
		if err := os.WriteFile(specPath, []byte(template), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "maslow init: failed to write %s: %v\n", specPath, err)
			return 1
		}
		fmt.Printf("maslow init: created %s\n", specPath)
	}

	return 0
}

func detectToolchainManager() string {
	// Check for mise (.mise.toml or .mise/*.toml)
	if fileExists(".mise.toml") || fileExists(".mise") {
		return "mise"
	}
	// Check for asdf (.tool-versions)
	if fileExists(".tool-versions") {
		return "asdf"
	}
	// Check for nix (flake.nix or shell.nix)
	if fileExists("flake.nix") || fileExists("shell.nix") {
		return "nix"
	}
	return ""
}

func cmdVersion() int {
	fmt.Printf("maslow %s (git: %s)\n", version, gitSHA)
	return 0
}

func printReport(report *evidence.Report, path string) {
	fmt.Printf("maslow: verdict=%s profile=%s\n", report.Verdict, report.Profile)
	for _, r := range report.CheckResults {
		fmt.Printf("  [%s] %s (%.2fs)\n", r.Status, r.Name, r.Duration)
		if r.Error != "" {
			fmt.Printf("         error: %s\n", r.Error)
		}
	}
	for _, c := range report.Contracts {
		fmt.Printf("  [%s] contract: %s\n", c.Status, c.Name)
		if c.Error != "" {
			fmt.Printf("         error: %s\n", c.Error)
		}
	}
	for _, b := range report.Budgets {
		if b.Actual != "" {
			fmt.Printf("  [%s] budget: %s (%s", b.Status, b.Name, b.Actual)
			if b.Limit != "" {
				fmt.Printf(" / %s", b.Limit)
			}
			fmt.Println(")")
		} else {
			fmt.Printf("  [%s] budget: %s\n", b.Status, b.Name)
		}
		if b.Error != "" {
			fmt.Printf("         error: %s\n", b.Error)
		}
	}
	for _, ref := range report.Refs {
		fmt.Printf("  [%s] ref: %s (%s)\n", ref.Status, ref.Path, ref.Kind)
		if ref.Error != "" {
			fmt.Printf("         error: %s\n", ref.Error)
		}
	}
	for _, p := range report.Policy {
		fmt.Printf("  [%s] policy/%s: %s\n", p.Status, p.Kind, p.Rule)
		if p.Detail != "" {
			fmt.Printf("         detail: %s\n", p.Detail)
		}
		if p.Error != "" {
			fmt.Printf("         error: %s\n", p.Error)
		}
	}
	fmt.Printf("maslow: report written to %s\n", path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
