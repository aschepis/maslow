package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// Decisions returned by Check.
const (
	DecisionPass        = "pass"
	DecisionNeedsReview = "needs_review"
	DecisionDecline     = "decline"
)

// Trusted is what the caller knows independently of the evidence file.
type Trusted struct {
	Head           string   // the commit the evidence must describe
	SpecSHA256     string   // hash of the trusted spec
	OverlaySHA256s []string // hashes of the overlays the run must have used, and no others
	Profile        string   // empty accepts any profile
	Base           string   // full merge-base SHA; empty skips the base_sha check
	// RequireRedGreen declines unless every overlay contract failed on Base and passed on Head.
	RequireRedGreen bool
}

// Decision is the outcome of checking an evidence file against trusted inputs.
type Decision struct {
	Decision string   `json:"decision"`
	Reasons  []Reason `json:"reasons"`
}

// Reason explains one mismatch, naming the evidence field it concerns.
type Reason struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FileSHA256 hashes a file's bytes the same way verify hashes the spec and overlays.
func FileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ReadReport loads an evidence file.
func ReadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence %s: %w", path, err)
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("failed to parse evidence %s: %w", path, err)
	}
	return &r, nil
}

// Check decides whether evidence approves a change. Any mismatch with the
// trusted inputs, or a fail or inconclusive verdict, declines it.
func Check(r *Report, t Trusted) Decision {
	var reasons []Reason
	add := func(field, format string, args ...any) {
		reasons = append(reasons, Reason{Field: field, Message: fmt.Sprintf(format, args...)})
	}

	if r.EvidenceVersion != Version {
		add("evidence_version", "evidence_version %q is not supported (want %q)", r.EvidenceVersion, Version)
	}
	if r.GitSHA != t.Head {
		add("git_sha", "git_sha %q does not match --head %q", r.GitSHA, t.Head)
	}
	if r.SpecSHA256 != t.SpecSHA256 {
		add("spec_sha256", "spec_sha256 %q does not match the trusted spec's %q", r.SpecSHA256, t.SpecSHA256)
	}
	for _, msg := range overlayMismatches(r.Overlays, t.OverlaySHA256s) {
		add("overlays", "%s", msg)
	}
	if t.Profile != "" && r.Profile != t.Profile {
		add("profile", "profile %q does not match --profile %q", r.Profile, t.Profile)
	}
	if t.Base != "" && r.BaseSHA != t.Base {
		add("base_sha", "base_sha %q does not match --base %q", r.BaseSHA, t.Base)
	}

	if r.Verdict != "pass" && r.Verdict != "needs_review" {
		add("verdict", "verdict %q does not approve the change", r.Verdict)
	}
	if t.RequireRedGreen {
		for _, msg := range redGreenMismatches(r, t.Base) {
			add("contracts", "%s", msg)
		}
	}

	if len(reasons) > 0 {
		return Decision{Decision: DecisionDecline, Reasons: reasons}
	}
	if r.Verdict == "needs_review" {
		return Decision{Decision: DecisionNeedsReview, Reasons: []Reason{{Field: "verdict", Message: "verdict is needs_review"}}}
	}
	return Decision{Decision: DecisionPass, Reasons: []Reason{}}
}

// redGreenMismatches returns one message per overlay contract that does not
// show red on base and green on the head, and per overlay with no results.
func redGreenMismatches(r *Report, base string) []string {
	if len(r.Overlays) == 0 {
		return []string{"--require-red-green but the run used no overlays"}
	}
	var msgs []string
	for _, o := range r.Overlays {
		if !hasOverlayContract(r.Contracts, o.Path) {
			msgs = append(msgs, fmt.Sprintf("overlay %s has no contract results", o.Path))
		}
	}
	for _, c := range r.Contracts {
		if c.Overlay == "" {
			continue
		}
		if problem := redGreenProblem(c, base); problem != "" {
			msgs = append(msgs, fmt.Sprintf("overlay contract %q (overlay %s) %s", c.Name, c.Overlay, problem))
		}
	}
	return msgs
}

func hasOverlayContract(contracts []ContractResult, overlay string) bool {
	for _, c := range contracts {
		if c.Overlay == overlay {
			return true
		}
	}
	return false
}

// Only the first problem is reported, so each contract yields at most one reason.
func redGreenProblem(c ContractResult, base string) string {
	switch {
	case c.Base == nil:
		return "has no base run; was verify run with --red-green?"
	case c.Base.SHA != base:
		return fmt.Sprintf("base ran on %q, not --base %q", c.Base.SHA, base)
	case c.Base.Status != "fail":
		return fmt.Sprintf("base status %q on %s: not red", c.Base.Status, c.Base.SHA)
	case c.HeadStatus != "pass":
		return fmt.Sprintf("head status %q: not green", c.HeadStatus)
	case c.Status != "pass":
		return fmt.Sprintf("status %q is not pass", c.Status)
	}
	return ""
}

func overlayMismatches(used []OverlayRef, trusted []string) []string {
	trustedSet := make(map[string]bool, len(trusted))
	for _, h := range trusted {
		trustedSet[h] = true
	}
	usedSet := make(map[string]bool, len(used))
	var msgs []string
	for _, o := range used {
		usedSet[o.SHA256] = true
		if !trustedSet[o.SHA256] {
			msgs = append(msgs, fmt.Sprintf("overlay %s (sha256 %s) is not trusted", o.Path, o.SHA256))
		}
	}
	for _, h := range trusted {
		if !usedSet[h] {
			msgs = append(msgs, fmt.Sprintf("trusted overlay sha256 %s was not used by the run", h))
		}
	}
	return msgs
}
