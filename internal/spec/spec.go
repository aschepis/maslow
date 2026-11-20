// Package spec handles loading and parsing maslow.yaml files.
package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MAS represents a parsed maslow.yaml file.
type MAS struct {
	MASVersion string             `yaml:"mas"`
	Project    Project            `yaml:"project"`
	Toolchain  *Toolchain         `yaml:"toolchain,omitempty"`
	Refs       []Ref              `yaml:"refs,omitempty"`
	Policy     *Policy            `yaml:"policy,omitempty"`
	Checks     *Checks            `yaml:"checks,omitempty"`
	Profiles   map[string]Profile `yaml:"profiles,omitempty"`
	Contracts  []Contract         `yaml:"contracts,omitempty"`
	Budgets    []Budget           `yaml:"budgets,omitempty"`
	Audit      *Audit             `yaml:"audit,omitempty"`
	Extensions map[string]any     `yaml:"extensions,omitempty"`
	Packages   []Package          `yaml:"packages,omitempty"`
	// MaxRetries caps every check's and contract's retries; nil means DefaultMaxRetries.
	MaxRetries *int `yaml:"max_retries,omitempty"`
}

// DefaultMaxRetries is the retries cap when a spec does not set max_retries.
const DefaultMaxRetries = 5

// RetryLimit is the most retries any check or contract in this spec may ask for.
func (m *MAS) RetryLimit() int {
	if m.MaxRetries == nil {
		return DefaultMaxRetries
	}
	return *m.MaxRetries
}

type Project struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
	License     string `yaml:"license,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
}

type Toolchain struct {
	Manager   string   `yaml:"manager,omitempty"`
	Tools     []Tool   `yaml:"tools,omitempty"`
	Lockfiles []string `yaml:"lockfiles,omitempty"`
}

type Tool struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Required bool   `yaml:"required"`
}

type Ref struct {
	Kind        string            `yaml:"kind"`
	Path        string            `yaml:"path"`
	Description string            `yaml:"description,omitempty"`
	Required    bool              `yaml:"required"`
	Name        string            `yaml:"name,omitempty"`
	Source      string            `yaml:"source,omitempty"`
	Transport   string            `yaml:"transport,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

type Policy struct {
	Deny      []string   `yaml:"deny,omitempty"`
	Allow     []string   `yaml:"allow,omitempty"`
	Protected []string   `yaml:"protected,omitempty"`
	Gated     []GatedDir `yaml:"gated,omitempty"`
	// ShrinkOnly files may lose lines in a change but never gain any.
	ShrinkOnly    []string       `yaml:"shrink_only,omitempty"`
	TestIntegrity *TestIntegrity `yaml:"test_integrity,omitempty"`
	// Observe holds rules reported without enforcing them; the schema forbids
	// an observe block inside another.
	Observe *Policy `yaml:"observe,omitempty"`
}

// TestIntegrity flags changes that weaken tests. With no Languages, every
// built-in language applies with its defaults.
type TestIntegrity struct {
	Languages map[string]TestLanguage `yaml:"languages,omitempty"`
	// Thresholds are files whose numbers are floors, such as coverage minimums.
	Thresholds []string `yaml:"thresholds,omitempty"`
}

// TestLanguage says which files hold tests and how to spot a test
// declaration and a skip marker. Tests and Skips are RE2 expressions.
type TestLanguage struct {
	Files []string `yaml:"files,omitempty"`
	Tests []string `yaml:"tests,omitempty"`
	Skips []string `yaml:"skips,omitempty"`
}

type GatedDir struct {
	Path  string `yaml:"path"`
	Owner string `yaml:"owner"`
}

type Checks struct {
	Runner []CheckRunner `yaml:"runner"`
}

type CheckRunner struct {
	Name    string   `yaml:"name"`
	Kind    string   `yaml:"kind"`
	Run     string   `yaml:"run"`
	Timeout string   `yaml:"timeout,omitempty"`
	Retries int      `yaml:"retries,omitempty"`
	Tags    []string `yaml:"tags,omitempty"`
}

// Profile selects what a run covers. A nil Contracts or Budgets list selects
// all of them; an empty list selects none. Refs and Policy default to true.
type Profile struct {
	Description string   `yaml:"description,omitempty"`
	Checks      []string `yaml:"checks"`
	Contracts   []string `yaml:"contracts,omitempty"`
	Budgets     []string `yaml:"budgets,omitempty"`
	Refs        *bool    `yaml:"refs,omitempty"`
	Policy      *bool    `yaml:"policy,omitempty"`
}

type Contract struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	// Env lists the environment variables templates may read as {{.env.NAME}}.
	Env []string `yaml:"env,omitempty"`
	// BaseURL resolves step URLs that start with "/"; an endpoint audit replaces it.
	BaseURL string `yaml:"base_url,omitempty"`
	// Retries reruns each failing scenario, with the contract's setup and teardown.
	Retries   int        `yaml:"retries,omitempty"`
	Setup     []Step     `yaml:"setup,omitempty"`
	Teardown  []Step     `yaml:"teardown,omitempty"`
	Scenarios []Scenario `yaml:"scenarios"`
}

type Scenario struct {
	Name     string `yaml:"name"`
	Setup    []Step `yaml:"setup,omitempty"`
	Teardown []Step `yaml:"teardown,omitempty"`
	Steps    []Step `yaml:"steps"`
	// Artifacts are repository-relative patterns for files the scenario leaves behind.
	Artifacts []string `yaml:"artifacts,omitempty"`
}

type Step struct {
	Action  string            `yaml:"action"`
	Method  string            `yaml:"method,omitempty"`
	URL     string            `yaml:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	// Multipart is sent as a multipart/form-data body instead of Body.
	Multipart []Part       `yaml:"multipart,omitempty"`
	Command   string       `yaml:"command,omitempty"`
	Args      []string     `yaml:"args,omitempty"`
	Stdin     string       `yaml:"stdin,omitempty"`
	Interval  string       `yaml:"interval,omitempty"`
	Timeout   string       `yaml:"timeout,omitempty"`
	Expect    *Expectation `yaml:"expect,omitempty"`
	From      string       `yaml:"from,omitempty"`
	As        string       `yaml:"as,omitempty"`
	Duration  string       `yaml:"duration,omitempty"`
	// Do is the browser operation: navigate, click, fill, wait or assert.
	Do       string `yaml:"do,omitempty"`
	Selector string `yaml:"selector,omitempty"`
	Text     string `yaml:"text,omitempty"`
	// Provider answers a metric, log or alerts step, such as prometheus.
	Provider string `yaml:"provider,omitempty"`
	Query    string `yaml:"query,omitempty"`
	// For is how long a metric, log or alerts step keeps polling; "" queries once.
	For string `yaml:"for,omitempty"`
	// Settle extends an alerts step's polling so slow alert rules can fire.
	Settle            string   `yaml:"settle,omitempty"`
	Matchers          []string `yaml:"matchers,omitempty"`
	IgnorePreexisting bool     `yaml:"ignore_preexisting,omitempty"`
}

// Part is one part of a multipart HTTP body. Exactly one of Value, File and
// Content is set: a text field, a fixture's bytes, or inline text sent as a file.
type Part struct {
	Field       string `yaml:"field"`
	Value       string `yaml:"value,omitempty"`
	File        string `yaml:"file,omitempty"`
	Content     string `yaml:"content,omitempty"`
	Filename    string `yaml:"filename,omitempty"`
	ContentType string `yaml:"content_type,omitempty"`
}

type Expectation struct {
	Status       *int              `yaml:"status,omitempty"`
	JSONPath     string            `yaml:"json_path,omitempty"`
	Value        any               `yaml:"value,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty"`
	BodyContains string            `yaml:"body_contains,omitempty"`
	BodyMatches  string            `yaml:"body_matches,omitempty"`
	ExitCode     *int              `yaml:"exit_code,omitempty"`
	Assertions   []Assertion       `yaml:"assertions,omitempty"`
	MinLength    *int              `yaml:"min_length,omitempty"`
	MaxLength    *int              `yaml:"max_length,omitempty"`
	Length       *int              `yaml:"length,omitempty"`
	Every        *Assertion        `yaml:"every,omitempty"`
	Some         *Assertion        `yaml:"some,omitempty"`
	// Browser assertions.
	TextContains string `yaml:"text_contains,omitempty"`
	URLContains  string `yaml:"url_contains,omitempty"`
	Visible      *bool  `yaml:"visible,omitempty"`
	Enabled      *bool  `yaml:"enabled,omitempty"`
	// Metric, log and alerts bounds, all inclusive.
	Min       *float64 `yaml:"min,omitempty"`
	Max       *float64 `yaml:"max,omitempty"`
	MinSeries *int     `yaml:"min_series,omitempty"`
	MaxSeries *int     `yaml:"max_series,omitempty"`
	// IfNoData is "pass" to accept an empty or NaN result; otherwise it fails.
	IfNoData string `yaml:"if_no_data,omitempty"`
}

type Assertion struct {
	JSONPath string `yaml:"json_path"`
	Value    any    `yaml:"value,omitempty"`
}

type Budget struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Tags         []string `yaml:"tags,omitempty"`
	Target       string   `yaml:"target"`
	P50          string   `yaml:"p50,omitempty"`
	P90          string   `yaml:"p90,omitempty"`
	P95          string   `yaml:"p95,omitempty"`
	P99          string   `yaml:"p99,omitempty"`
	Max          string   `yaml:"max,omitempty"`
	ErrorRate    *float64 `yaml:"error_rate,omitempty"`
	Concurrency  *int     `yaml:"concurrency,omitempty"`
	Duration     string   `yaml:"duration,omitempty"`
	Warmup       string   `yaml:"warmup,omitempty"`
	Uptime       *float64 `yaml:"uptime,omitempty"`
	MaxRPS       *int     `yaml:"max_rps,omitempty"`
	Burst        *int     `yaml:"burst,omitempty"`
	MaxLines     *int     `yaml:"max_lines,omitempty"`
	MaxFunctions *int     `yaml:"max_functions,omitempty"`
	MaxDepth     *int     `yaml:"max_depth,omitempty"`
}

type Audit struct {
	Targets []AuditTarget `yaml:"targets"`
}

type AuditTarget struct {
	Kind    string            `yaml:"kind"`
	Path    string            `yaml:"path"`
	Env     map[string]string `yaml:"env,omitempty"`
	Timeout string            `yaml:"timeout,omitempty"`
	Retries *int              `yaml:"retries,omitempty"`
	Checks  []string          `yaml:"checks,omitempty"`
}

type Package struct {
	Name    string   `yaml:"name"`
	Path    string   `yaml:"path"`
	Checks  []string `yaml:"checks,omitempty"`
	Budgets []string `yaml:"budgets,omitempty"`
}

// Load reads and parses a maslow.yaml file.
func Load(path string) (*MAS, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return Parse(data, path)
}

// Parse parses maslow.yaml content; name is used in error messages.
func Parse(data []byte, name string) (*MAS, error) {
	var mas MAS
	if err := yaml.Unmarshal(data, &mas); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", name, err)
	}
	return &mas, nil
}
