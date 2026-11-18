package runner

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/spec"
)

// maxReportedLines keeps one rule from flooding the evidence when, say, a
// whole new file lands in a shrink-only path.
const maxReportedLines = 100

// runLineRules evaluates the rules that need line-level changes, not just paths.
func runLineRules(policy *spec.Policy, scope PolicyScope) []evidence.PolicyResult {
	if len(policy.ShrinkOnly) == 0 && policy.TestIntegrity == nil {
		return nil
	}

	diffs, err := gitrepo.DiffLines(scope.Dir, scope.BaseSHA)
	if err != nil {
		results := ruleErrors("shrink_only", policy.ShrinkOnly, err)
		if policy.TestIntegrity != nil {
			results = append(results, testIntegrityError(err))
		}
		return results
	}

	results := checkShrinkOnly(policy.ShrinkOnly, diffs)
	if policy.TestIntegrity != nil {
		results = append(results, checkTestIntegrity(policy.TestIntegrity, diffs)...)
	}
	return results
}

// checkShrinkOnly fails a pattern when any matching file gained a line.
// Editing a line counts, since the new version of it is an added line.
func checkShrinkOnly(patterns []string, diffs []gitrepo.FileDiff) []evidence.PolicyResult {
	results := make([]evidence.PolicyResult, 0, len(patterns))
	for _, pattern := range patterns {
		result := evidence.PolicyResult{Rule: pattern, Kind: "shrink_only"}
		if err := validatePattern(pattern); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("policy.shrink_only: invalid pattern: %v", err)
			results = append(results, result)
			continue
		}

		var files, lines []string
		for _, d := range diffs {
			matched, _ := matchPath(pattern, d.Path)
			added := addedLines(d)
			if !matched || len(added) == 0 {
				continue
			}
			files = append(files, d.Path)
			lines = append(lines, located(d.Path, added)...)
		}

		if len(files) == 0 {
			result.Status = "pass"
		} else {
			result.Status = "fail"
			result.Detail = fmt.Sprintf("%d line(s) added to shrink-only file(s): %s", len(lines), strings.Join(files, ", "))
			result.Files = files
			result.Lines = capLines(lines)
		}
		results = append(results, result)
	}
	return results
}

func addedLines(d gitrepo.FileDiff) []gitrepo.Line {
	var added []gitrepo.Line
	for _, h := range d.Hunks {
		added = append(added, h.Added...)
	}
	return added
}

func located(path string, lines []gitrepo.Line) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, fmt.Sprintf("%s:%d: %s", path, l.Number, strings.TrimSpace(l.Text)))
	}
	return out
}

func capLines(lines []string) []string {
	if len(lines) <= maxReportedLines {
		return lines
	}
	capped := append([]string{}, lines[:maxReportedLines]...)
	return append(capped, fmt.Sprintf("... %d more line(s) not shown", len(lines)-maxReportedLines))
}

// builtinTestLanguages are the defaults behind policy.test_integrity.languages.
var builtinTestLanguages = map[string]spec.TestLanguage{
	"go": {
		Files: []string{"**/*_test.go"},
		Tests: []string{`^func\s+(\([^)]*\)\s*)?(Test|Benchmark|Fuzz|Example)\w*\(`},
		Skips: []string{`\.Skip(Now|f)?\(`},
	},
	"ruby": {
		Files: []string{"**/*_spec.rb", "**/*_test.rb", "**/test_*.rb"},
		Tests: []string{`^\s*(it|specify|example|scenario|test)\b\s*\(?\s*["'][^"']*["']`, `^\s*def\s+test_\w+`},
		Skips: []string{`^\s*x(it|describe|context|specify|example|scenario|feature)\b`, `^\s*(skip|pending)\b`, `\bskip:\s*true\b`},
	},
	"javascript": {
		Files: javascriptTestFiles(),
		Tests: []string{"^\\s*(it|test)\\s*\\(\\s*[\"'`][^\"'`]*[\"'`]"},
		// .only silently skips every sibling test, so it counts as a skip.
		Skips: []string{`\b(it|test|describe|context|suite)\.(skip|todo|only)\b`, `^\s*x(it|test|describe)\b`},
	},
	"python": {
		Files: []string{"**/test_*.py", "**/*_test.py"},
		Tests: []string{`^\s*(async\s+)?def\s+test\w*`},
		Skips: []string{`@pytest\.mark\.(skip|skipif|xfail)\b`, `@unittest\.(skip|skipIf|skipUnless|expectedFailure)\b`, `\bpytest\.skip\(`, `\.skipTest\(`},
	},
}

func javascriptTestFiles() []string {
	files := []string{"**/__tests__/**"}
	for _, ext := range []string{"js", "jsx", "ts", "tsx", "mjs", "cjs"} {
		files = append(files, "**/*.test."+ext, "**/*.spec."+ext)
	}
	return files
}

type testLanguage struct {
	files []string
	tests []*regexp.Regexp
	skips []*regexp.Regexp
}

// resolveTestLanguages applies built-in defaults to the configured languages
// and compiles them, naming the spec section of anything invalid.
func resolveTestLanguages(cfg *spec.TestIntegrity) ([]testLanguage, error) {
	configured := cfg.Languages
	if len(configured) == 0 {
		configured = make(map[string]spec.TestLanguage, len(builtinTestLanguages))
		for name := range builtinTestLanguages {
			configured[name] = spec.TestLanguage{}
		}
	}

	names := make([]string, 0, len(configured))
	for name := range configured {
		names = append(names, name)
	}
	sort.Strings(names)

	langs := make([]testLanguage, 0, len(names))
	for _, name := range names {
		section := "policy.test_integrity.languages." + name
		merged := withDefaults(configured[name], builtinTestLanguages[name])
		if len(merged.Files) == 0 {
			return nil, fmt.Errorf("%s.files: required for a language without built-in defaults", section)
		}
		for _, pattern := range merged.Files {
			if err := validatePattern(pattern); err != nil {
				return nil, fmt.Errorf("%s.files: %v", section, err)
			}
		}
		tests, err := compileAll(section+".tests", merged.Tests)
		if err != nil {
			return nil, err
		}
		skips, err := compileAll(section+".skips", merged.Skips)
		if err != nil {
			return nil, err
		}
		langs = append(langs, testLanguage{files: merged.Files, tests: tests, skips: skips})
	}
	return langs, nil
}

func withDefaults(lang, defaults spec.TestLanguage) spec.TestLanguage {
	if len(lang.Files) == 0 {
		lang.Files = defaults.Files
	}
	if len(lang.Tests) == 0 {
		lang.Tests = defaults.Tests
	}
	if len(lang.Skips) == 0 {
		lang.Skips = defaults.Skips
	}
	return lang
}

func compileAll(section string, exprs []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(exprs))
	for _, expr := range exprs {
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid regexp %q: %v", section, expr, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

// marker is a test declaration or skip found on a changed line. Key is what
// a matching marker on the other side of the change must share to cancel it.
type marker struct {
	location string
	file     string
	key      string
}

// checkTestIntegrity reports each way a change can weaken tests as its own
// result, so evidence shows which kind of weakening happened.
func checkTestIntegrity(cfg *spec.TestIntegrity, diffs []gitrepo.FileDiff) []evidence.PolicyResult {
	langs, err := resolveTestLanguages(cfg)
	if err == nil {
		err = validatePatterns("policy.test_integrity.thresholds", cfg.Thresholds)
	}
	if err != nil {
		return []evidence.PolicyResult{testIntegrityError(err)}
	}

	var deleted []string
	var removedTests, addedTests, removedSkips, addedSkips []marker
	var lowered, loweredFiles []string
	for _, d := range diffs {
		var tests, skips []*regexp.Regexp
		for _, lang := range langs {
			if matchesAny(lang.files, d.Path) {
				tests = append(tests, lang.tests...)
				skips = append(skips, lang.skips...)
			}
		}
		if len(tests) > 0 || len(skips) > 0 {
			if d.Deleted {
				deleted = append(deleted, d.Path)
			}
			for _, h := range d.Hunks {
				removedTests = append(removedTests, testMarkers(d.Path, h.Removed, tests)...)
				addedTests = append(addedTests, testMarkers(d.Path, h.Added, tests)...)
				removedSkips = append(removedSkips, skipMarkers(d.Path, h.Removed, skips)...)
				addedSkips = append(addedSkips, skipMarkers(d.Path, h.Added, skips)...)
			}
		}
		if matchesAny(cfg.Thresholds, d.Path) {
			if lines := loweredNumbers(d); len(lines) > 0 {
				lowered = append(lowered, lines...)
				loweredFiles = append(loweredFiles, d.Path)
			}
		}
	}

	results := []evidence.PolicyResult{
		reviewResult("deleted_test_files", fmt.Sprintf("%d test file(s) deleted", len(deleted)), deleted, nil),
		markerResult("removed_tests", "test(s) removed and not added back elsewhere", uncancelled(removedTests, addedTests)),
		markerResult("added_skips", "skip marker(s) added", uncancelled(addedSkips, removedSkips)),
	}
	if len(cfg.Thresholds) > 0 {
		results = append(results, reviewResult("lowered_thresholds", fmt.Sprintf("%d threshold(s) lowered", len(lowered)), loweredFiles, lowered))
	}
	return results
}

func testIntegrityError(err error) evidence.PolicyResult {
	return evidence.PolicyResult{Rule: "test_integrity", Kind: "test_integrity", Status: "error", Error: err.Error()}
}

func validatePatterns(section string, patterns []string) error {
	for _, pattern := range patterns {
		if err := validatePattern(pattern); err != nil {
			return fmt.Errorf("%s: %v", section, err)
		}
	}
	return nil
}

func matchesAny(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if ok, _ := matchPath(pattern, path); ok {
			return true
		}
	}
	return false
}

// testMarkers keys a declaration by the matched text, which names the test,
// so a test whose signature line is edited or moved is not reported removed.
func testMarkers(path string, lines []gitrepo.Line, exprs []*regexp.Regexp) []marker {
	var found []marker
	for _, l := range lines {
		for _, re := range exprs {
			if m := re.FindString(l.Text); m != "" {
				found = append(found, marker{location: located(path, []gitrepo.Line{l})[0], file: path, key: strings.TrimSpace(m)})
				break
			}
		}
	}
	return found
}

// skipMarkers keys a skip by its whole line, so only an identical skip line
// removed elsewhere cancels it.
func skipMarkers(path string, lines []gitrepo.Line, exprs []*regexp.Regexp) []marker {
	var found []marker
	for _, l := range lines {
		for _, re := range exprs {
			if re.MatchString(l.Text) {
				found = append(found, marker{location: located(path, []gitrepo.Line{l})[0], file: path, key: strings.TrimSpace(l.Text)})
				break
			}
		}
	}
	return found
}

func uncancelled(markers, cancelling []marker) []marker {
	keys := make(map[string]bool, len(cancelling))
	for _, m := range cancelling {
		keys[m.key] = true
	}
	var left []marker
	for _, m := range markers {
		if !keys[m.key] {
			left = append(left, m)
		}
	}
	return left
}

func markerResult(rule, what string, markers []marker) evidence.PolicyResult {
	var files, lines []string
	for _, m := range markers {
		if len(files) == 0 || files[len(files)-1] != m.file {
			files = append(files, m.file)
		}
		lines = append(lines, m.location)
	}
	return reviewResult(rule, fmt.Sprintf("%d %s", len(markers), what), files, lines)
}

// reviewResult asks for review when anything was found; summary leads the detail.
func reviewResult(rule, summary string, files, lines []string) evidence.PolicyResult {
	result := evidence.PolicyResult{Rule: rule, Kind: "test_integrity"}
	if len(files) == 0 {
		result.Status = "pass"
		return result
	}
	result.Status = "needs_review"
	result.Detail = fmt.Sprintf("%s: %s", summary, strings.Join(files, ", "))
	result.Files = files
	result.Lines = capLines(lines)
	return result
}

var number = regexp.MustCompile(`\d+(\.\d+)?`)

// loweredNumbers pairs each removed line with the added line in its place
// and reports pairs that differ only in numbers, with at least one lower.
func loweredNumbers(d gitrepo.FileDiff) []string {
	var out []string
	for _, h := range d.Hunks {
		for i := 0; i < len(h.Removed) && i < len(h.Added); i++ {
			before, after := h.Removed[i], h.Added[i]
			if lowersANumber(before.Text, after.Text) {
				out = append(out, fmt.Sprintf("%s:%d: %s -> %s", d.Path, after.Number, strings.TrimSpace(before.Text), strings.TrimSpace(after.Text)))
			}
		}
	}
	return out
}

func lowersANumber(before, after string) bool {
	if number.ReplaceAllString(before, "#") != number.ReplaceAllString(after, "#") {
		return false
	}
	beforeNums, afterNums := number.FindAllString(before, -1), number.FindAllString(after, -1)
	for i := range beforeNums {
		b, _ := strconv.ParseFloat(beforeNums[i], 64)
		a, _ := strconv.ParseFloat(afterNums[i], 64)
		if a < b {
			return true
		}
	}
	return false
}
