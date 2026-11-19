package evidence

import (
	"fmt"
	"strings"
)

// Summary renders a report as the Markdown the GitHub Action writes to its
// job summary: the verdict, the enforced results that failed or need review,
// and the observe-mode policy findings, which never affect the verdict.
func Summary(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Maslow: `%s`\n", r.Verdict)
	fmt.Fprintf(&b, "%s\n\n", metadataLine(r))
	writeEnforced(&b, r)
	writeObserved(&b, r)
	return b.String()
}

func metadataLine(r *Report) string {
	line := fmt.Sprintf("Profile `%s` · spec `%s` from `%s` · maslow %s", r.Profile, r.SpecPath, r.SpecSource, r.MaslowVersion)
	if r.BaseSHA != "" {
		line += fmt.Sprintf(" · base `%s`", shortSHA(r.BaseSHA))
	}
	return line
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func writeEnforced(b *strings.Builder, r *Report) {
	var rows [][]string
	for _, c := range r.CheckResults {
		if isBad(c.Status) {
			rows = append(rows, []string{"check", c.Name, c.Status, c.Error})
		}
	}
	for _, c := range r.Contracts {
		if isBad(c.Status) {
			rows = append(rows, []string{"contract", c.Name, c.Status, c.Error})
		}
	}
	for _, bg := range r.Budgets {
		if isBad(bg.Status) {
			rows = append(rows, []string{"budget", bg.Name, bg.Status, firstNonEmpty(bg.Error, bg.Actual)})
		}
	}
	for _, ref := range r.Refs {
		if isBad(ref.Status) {
			rows = append(rows, []string{"ref", ref.Path, ref.Status, ref.Error})
		}
	}
	for _, p := range r.Policy {
		if p.Mode != ModeObserve && (isBad(p.Status) || p.Status == "needs_review") {
			rows = append(rows, []string{"policy/" + p.Kind, p.Rule, p.Status, firstNonEmpty(p.Detail, p.Error)})
		}
	}

	if len(rows) == 0 {
		b.WriteString("Nothing failed.\n")
		return
	}
	writeTable(b, []string{"Section", "Item", "Status", "Detail"}, rows)
}

func writeObserved(b *strings.Builder, r *Report) {
	var rows [][]string
	observed, passed := 0, 0
	for _, p := range r.Policy {
		if p.Mode != ModeObserve {
			continue
		}
		observed++
		switch {
		case strings.HasPrefix(p.Status, "would_"):
			rows = append(rows, []string{p.Kind, p.Rule, p.Status, observedDetail(p)})
		case p.Status == "pass":
			passed++
		}
	}
	if observed == 0 {
		return
	}

	b.WriteString("\n### Observed policy rules (not enforced)\n\n")
	b.WriteString("These rules run in observe mode. Their findings do not change the verdict.\n")
	if len(rows) > 0 {
		b.WriteString("\n")
		writeTable(b, []string{"Kind", "Rule", "Would be", "Detail"}, rows)
	}
	if passed == 1 {
		b.WriteString("\n1 observed rule passed.\n")
	}
	if passed > 1 {
		fmt.Fprintf(b, "\n%d observed rules passed.\n", passed)
	}
}

func observedDetail(p PolicyResult) string {
	if p.Detail != "" {
		return p.Detail
	}
	if len(p.Files) > 0 {
		return p.Files[0]
	}
	return p.Error
}

// isBad matches the statuses that make the verdict fail; not_red is one.
func isBad(status string) bool {
	return status == "fail" || status == "error" || status == StatusNotRed
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func writeTable(b *strings.Builder, header []string, rows [][]string) {
	fmt.Fprintf(b, "| %s |\n", strings.Join(header, " | "))
	b.WriteString(strings.Repeat("|---", len(header)) + "|\n")
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, c := range row {
			cells[i] = cell(c)
		}
		fmt.Fprintf(b, "| %s |\n", strings.Join(cells, " | "))
	}
}

// cell keeps a value on one table row: newlines would end the row and a bare
// pipe would start a new column.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "|", `\|`)
}
