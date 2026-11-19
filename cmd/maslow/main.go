// Maslow CLI — verifies a repository against a declarative spec and writes evidence.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aschepis/maslow/internal/audit"
	"github.com/aschepis/maslow/internal/buildinfo"
	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/schema"
	"github.com/aschepis/maslow/internal/spec"
	"github.com/aschepis/maslow/internal/verify"
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
	case "evidence":
		os.Exit(cmdEvidence(os.Args[2:], os.Stdout))
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
  verify   --profile <name>    Run verification and write evidence
           [--base <ref>]      Judge policy on every change since the merge base with <ref>
           [--spec-from base]  Verify against the spec at the merge base, not the branch's
           [--overlay <file>]  Also run the contracts in <file>; repeatable
           [--red-green]       Fail overlay contracts that already pass on the merge base (needs --base)
           [--report <path>]   Evidence file (default reports/verify.json)
           [--output-limit n]  Bytes of each check's output to keep (default 4096)
           [--strict]          Exit 1 on needs_review too (default: only fail and inconclusive)
  audit    --profile <name>    Run black-box audit (default report reports/audit.json)
  evidence check <file>        Decide pass, needs_review or decline for an evidence file
           --head <sha>        Commit the evidence must describe
           --spec <file>       Trusted spec whose hash the evidence must carry
           [--profile <name>]  Profile the evidence must have run
           [--overlay <file>]  Trusted overlay the run must have used; repeatable
           [--overlay-sha256 <hex>]  Same, given as a hash; repeatable
           [--base <sha>]      Full merge-base SHA the evidence's base_sha must match
           [--require-red-green]  Decline unless every overlay contract failed on --base
                               and passed on --head (needs --base and an overlay flag)
           [--json]          Print the decision and reasons as JSON
  evidence summary <file>      Print an evidence file as the Markdown job summary
  init    [--apply]           Add a starter maslow.yaml to the current directory
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

// stringList collects a flag given more than once.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

// newFlagSet returns a flag set that reports usage errors without exiting.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("maslow "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func cmdVerify(args []string) int {
	fs := newFlagSet("verify")
	profile := fs.String("profile", "quick", "profile to run")
	specPath := fs.String("spec", "maslow.yaml", "path to the spec, relative to the repository root")
	base := fs.String("base", "", "git ref; judge policy on every change since its merge base with HEAD")
	specFrom := fs.String("spec-from", "head", "read the spec from head (the working tree) or base (the merge base)")
	reportPath := fs.String("report", verify.DefaultReportPath, "where to write the evidence file")
	outputLimit := fs.Int("output-limit", evidence.DefaultOutputLimit, "bytes of each check's output to keep (0 keeps everything)")
	strict := fs.Bool("strict", false, "exit non-zero on needs_review as well as fail and inconclusive")
	redGreen := fs.Bool("red-green", false, "also run overlay contracts on the merge base; fail any that already pass there")
	var overlayPaths stringList
	fs.Var(&overlayPaths, "overlay", "file of extra contracts to run beside the spec's (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "maslow verify: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	if *specFrom != "head" && *specFrom != "base" {
		fmt.Fprintf(os.Stderr, "maslow verify: --spec-from must be head or base, got %q\n", *specFrom)
		return 2
	}
	if *specFrom == "base" && *base == "" {
		fmt.Fprintln(os.Stderr, "maslow verify: --spec-from base requires --base <ref>")
		return 2
	}

	if *redGreen && (*base == "" || len(overlayPaths) == 0) {
		fmt.Fprintln(os.Stderr, "maslow verify: --red-green requires --base <ref> and at least one --overlay <file>")
		return 2
	}

	opts := verify.Options{Profile: *profile, SpecPath: *specPath, SpecFromBase: *specFrom == "base", RedGreen: *redGreen}
	if *base != "" {
		sha, err := verify.ResolveBase(*base)
		if err != nil {
			fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
			return 1
		}
		opts.BaseSHA = sha
	}

	loaded, err := verify.ReadSpec(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		if opts.SpecFromBase {
			return finish("verify", verify.SpecLoadFailure(opts, err), *reportPath, *outputLimit, *strict)
		}
		return 1
	}
	if opts.SpecAddedOnBranch {
		fmt.Fprintf(os.Stderr, "maslow verify: %s is not in base %s; verifying against the branch's spec, which needs review\n", *specPath, opts.BaseSHA)
	}
	opts.SpecSHA256 = loaded.SHA256
	if _, ok := loaded.MAS.Profiles[*profile]; !ok {
		fmt.Fprintf(os.Stderr, "maslow verify: profile %q not found in %s\n", *profile, *specPath)
		return 2
	}
	for _, path := range overlayPaths {
		overlay, err := spec.ReadOverlay(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
			return 1
		}
		opts.Overlays = append(opts.Overlays, overlay)
	}

	report, err := verify.Run(loaded.MAS, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow verify: %v\n", err)
		return 1
	}

	return finish("verify", report, *reportPath, *outputLimit, *strict)
}

func cmdAudit(args []string) int {
	fs := newFlagSet("audit")
	profile := fs.String("profile", "full", "profile to record in the evidence")
	specPath := fs.String("spec", "maslow.yaml", "path to the spec")
	reportPath := fs.String("report", audit.DefaultReportPath, "where to write the evidence file")
	outputLimit := fs.Int("output-limit", evidence.DefaultOutputLimit, "bytes of each check's output to keep (0 keeps everything)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "maslow audit: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	loaded, err := spec.Read(*specPath, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}

	report, err := audit.Run(loaded.MAS, audit.Options{Profile: *profile, SpecPath: *specPath, SpecSHA256: loaded.SHA256})
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow audit: %v\n", err)
		return 1
	}

	return finish("audit", report, *reportPath, *outputLimit, false)
}

// finish caps output, writes the evidence, prints a summary and returns the exit code.
func finish(command string, report *evidence.Report, reportPath string, outputLimit int, strict bool) int {
	report.CapOutput(outputLimit)
	if err := report.Write(reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "maslow %s: %v\n", command, err)
		return 1
	}

	printReport(report, reportPath)
	return exitCode(report.Verdict, strict)
}

// exitCode lets needs_review through unless strict, so a pipeline can route
// the change to a person by reading the verdict instead of failing the job.
func exitCode(verdict string, strict bool) int {
	switch verdict {
	case "pass":
		return 0
	case "needs_review":
		if strict {
			return 1
		}
		return 0
	default:
		return 1
	}
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
	fmt.Printf("maslow %s (git: %s)\n", buildinfo.Version(), buildinfo.GitSHA())
	return 0
}

func printReport(report *evidence.Report, path string) {
	fmt.Printf("maslow: verdict=%s profile=%s\n", report.Verdict, report.Profile)
	if report.BaseSHA != "" {
		fmt.Printf("maslow: policy judged against base %s\n", report.BaseSHA)
	}
	for _, r := range report.CheckResults {
		fmt.Printf("  [%s] %s (%.2fs)\n", r.Status, r.Name, r.Duration)
		if len(r.Attempts) > 0 {
			fmt.Printf("         attempts: %d\n", len(r.Attempts))
		}
		if r.Error != "" {
			fmt.Printf("         error: %s\n", r.Error)
		}
	}
	for _, c := range report.Contracts {
		if c.Overlay != "" {
			fmt.Printf("  [%s] contract: %s (overlay %s)\n", c.Status, c.Name, c.Overlay)
		} else {
			fmt.Printf("  [%s] contract: %s\n", c.Status, c.Name)
		}
		if len(c.Attempts) > 0 {
			fmt.Printf("         attempts: %d\n", len(c.Attempts))
		}
		if f := c.FailedStep; f != nil {
			fmt.Printf("         step: %s\n", strings.TrimSpace(fmt.Sprintf("%s[%d] %s %s", f.Phase, f.Index, f.Action, f.Target)))
		}
		if c.Error != "" {
			fmt.Printf("         error: %s\n", c.Error)
		}
		if b := c.Base; b != nil {
			fmt.Printf("         base %s: %s, head: %s\n", b.SHA, b.Status, c.HeadStatus)
			if b.Error != "" {
				fmt.Printf("         base error: %s\n", b.Error)
			}
		}
		for _, a := range c.Artifacts {
			fmt.Printf("         artifact: %s\n", a)
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
		mode := ""
		if p.Mode != "" {
			mode = " (" + p.Mode + ")"
		}
		fmt.Printf("  [%s] policy/%s%s: %s\n", p.Status, p.Kind, mode, p.Rule)
		if p.Owner != "" {
			fmt.Printf("         owner: %s\n", p.Owner)
		}
		if p.Detail != "" {
			fmt.Printf("         detail: %s\n", p.Detail)
		}
		for _, line := range p.Lines {
			fmt.Printf("         line: %s\n", line)
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
