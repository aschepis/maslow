// Package spec handles loading and parsing maslow.yaml files.
package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MAS represents a parsed maslow.yaml file.
type MAS struct {
	MASVersion string      `yaml:"mas"`
	Project    Project     `yaml:"project"`
	Toolchain  *Toolchain  `yaml:"toolchain,omitempty"`
	Refs       []Ref       `yaml:"refs,omitempty"`
	Policy     *Policy     `yaml:"policy,omitempty"`
	Checks     *Checks     `yaml:"checks,omitempty"`
	Profiles   map[string]Profile `yaml:"profiles,omitempty"`
	Contracts  []Contract  `yaml:"contracts,omitempty"`
	Budgets    []Budget    `yaml:"budgets,omitempty"`
	Audit      *Audit      `yaml:"audit,omitempty"`
	Extensions map[string]any `yaml:"extensions,omitempty"`
	Packages   []Package   `yaml:"packages,omitempty"`
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
	Tags    []string `yaml:"tags,omitempty"`
}

type Profile struct {
	Description string   `yaml:"description,omitempty"`
	Checks      []string `yaml:"checks"`
}

type Contract struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Setup       []Step     `yaml:"setup,omitempty"`
	Teardown    []Step     `yaml:"teardown,omitempty"`
	Scenarios   []Scenario `yaml:"scenarios"`
}

type Scenario struct {
	Name     string `yaml:"name"`
	Setup    []Step `yaml:"setup,omitempty"`
	Teardown []Step `yaml:"teardown,omitempty"`
	Steps    []Step `yaml:"steps"`
}

type Step struct {
	Action   string            `yaml:"action"`
	Method   string            `yaml:"method,omitempty"`
	URL      string            `yaml:"url,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Body     string            `yaml:"body,omitempty"`
	Command  string            `yaml:"command,omitempty"`
	Args     []string          `yaml:"args,omitempty"`
	Stdin    string            `yaml:"stdin,omitempty"`
	Interval string            `yaml:"interval,omitempty"`
	Timeout  string            `yaml:"timeout,omitempty"`
	Expect   *Expectation      `yaml:"expect,omitempty"`
	From     string            `yaml:"from,omitempty"`
	As       string            `yaml:"as,omitempty"`
	Duration string            `yaml:"duration,omitempty"`
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
}

type Assertion struct {
	JSONPath string `yaml:"json_path"`
	Value    any    `yaml:"value,omitempty"`
}

type Budget struct {
	Name         string  `yaml:"name"`
	Kind         string  `yaml:"kind"`
	Target       string  `yaml:"target"`
	P50          string  `yaml:"p50,omitempty"`
	P90          string  `yaml:"p90,omitempty"`
	P95          string  `yaml:"p95,omitempty"`
	P99          string  `yaml:"p99,omitempty"`
	Max          string  `yaml:"max,omitempty"`
	ErrorRate    *float64 `yaml:"error_rate,omitempty"`
	Concurrency  *int    `yaml:"concurrency,omitempty"`
	Duration     string  `yaml:"duration,omitempty"`
	Warmup       string  `yaml:"warmup,omitempty"`
	Uptime       *float64 `yaml:"uptime,omitempty"`
	MaxRPS       *int    `yaml:"max_rps,omitempty"`
	Burst        *int    `yaml:"burst,omitempty"`
	MaxLines     *int    `yaml:"max_lines,omitempty"`
	MaxFunctions *int    `yaml:"max_functions,omitempty"`
	MaxDepth     *int    `yaml:"max_depth,omitempty"`
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

	var mas MAS
	if err := yaml.Unmarshal(data, &mas); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &mas, nil
}
