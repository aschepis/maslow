package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aschepis/maslow/internal/evidence"
)

func cmdEvidence(args []string, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "check" {
		return cmdEvidenceCheck(args[1:], stdout)
	}
	if len(args) > 0 && args[0] == "summary" {
		return cmdEvidenceSummary(args[1:], stdout)
	}
	fmt.Fprintln(os.Stderr, "Usage: maslow evidence check <file> --head <sha> --spec <file> [--profile <name>] [--overlay <file>] [--overlay-sha256 <hex>] [--base <sha>] [--require-red-green] [--json]")
	fmt.Fprintln(os.Stderr, "       maslow evidence summary <file>")
	return 2
}

func cmdEvidenceSummary(args []string, stdout io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "maslow evidence summary: expected exactly one evidence file argument")
		return 2
	}
	report, err := evidence.ReadReport(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "maslow evidence summary: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, evidence.Summary(report))
	return 0
}

func cmdEvidenceCheck(args []string, stdout io.Writer) int {
	fs := newFlagSet("evidence check")
	head := fs.String("head", "", "commit the evidence must describe")
	specPath := fs.String("spec", "", "trusted spec the evidence must have been produced from")
	profile := fs.String("profile", "", "profile the evidence must have run (default: any)")
	base := fs.String("base", "", "full merge-base SHA the evidence's base_sha must match")
	requireRedGreen := fs.Bool("require-red-green", false, "decline unless every overlay contract failed on --base and passed on --head")
	asJSON := fs.Bool("json", false, "print the decision and reasons as JSON")
	var overlayPaths, overlayHashes stringList
	fs.Var(&overlayPaths, "overlay", "trusted overlay file the run must have used (repeatable)")
	fs.Var(&overlayHashes, "overlay-sha256", "sha256 of a trusted overlay the run must have used (repeatable)")

	positional, ok := parseInterspersed(fs, args)
	if !ok {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "maslow evidence check: expected exactly one evidence file argument")
		return 2
	}
	if *head == "" || *specPath == "" {
		fmt.Fprintln(os.Stderr, "maslow evidence check: --head and --spec are required")
		return 2
	}
	if *requireRedGreen && *base == "" {
		fmt.Fprintln(os.Stderr, "maslow evidence check: --require-red-green requires --base <sha>")
		return 2
	}
	if *requireRedGreen && len(overlayPaths) == 0 && len(overlayHashes) == 0 {
		fmt.Fprintln(os.Stderr, "maslow evidence check: --require-red-green requires --overlay or --overlay-sha256")
		return 2
	}

	trusted := evidence.Trusted{
		Head:            *head,
		Profile:         *profile,
		OverlaySHA256s:  overlayHashes,
		Base:            *base,
		RequireRedGreen: *requireRedGreen,
	}
	var err error
	if trusted.SpecSHA256, err = evidence.FileSHA256(*specPath); err != nil {
		fmt.Fprintf(os.Stderr, "maslow evidence check: trusted spec: %v\n", err)
		return 2
	}
	for _, path := range overlayPaths {
		h, err := evidence.FileSHA256(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "maslow evidence check: trusted overlay: %v\n", err)
			return 2
		}
		trusted.OverlaySHA256s = append(trusted.OverlaySHA256s, h)
	}

	path := positional[0]
	var decision evidence.Decision
	if report, err := evidence.ReadReport(path); err != nil {
		decision = evidence.Decision{Decision: evidence.DecisionDecline, Reasons: []evidence.Reason{{Field: "evidence", Message: err.Error()}}}
	} else {
		decision = evidence.Check(report, trusted)
	}

	if *asJSON {
		printDecisionJSON(stdout, path, decision)
	} else {
		fmt.Fprintf(stdout, "maslow evidence check: %s\n", decision.Decision)
		for _, r := range decision.Reasons {
			fmt.Fprintf(stdout, "  %s: %s\n", path, r.Message)
		}
	}

	if decision.Decision == evidence.DecisionPass {
		return 0
	}
	return 1
}

func printDecisionJSON(w io.Writer, path string, d evidence.Decision) {
	out := struct {
		Evidence string `json:"evidence"`
		evidence.Decision
	}{path, d}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(w, string(data))
}

// parseInterspersed lets flags follow positional arguments, which the flag
// package alone stops parsing at.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, bool) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, true
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}
