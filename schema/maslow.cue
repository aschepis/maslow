// Maslow Agent Spec (MAS) Schema
// Validates maslow.yaml structure and constraints.

package maslow

// Top-level MAS document.
#MAS: {
	// MAS schema version (required).
	mas: =~"^[0-9]+\\.[0-9]+$"

	// Project metadata.
	project: #Project

	// Toolchain requirements.
	toolchain?: #Toolchain

	// External references.
	refs?: [...#Ref]

	// Policy constraints.
	policy?: #Policy

	// The most retries any check or contract may ask for.
	max_retries: *5 | int & >=0

	// Check runners.
	checks?: #Checks & {runner: [...{retries?: <=max_retries}]}

	// Profiles select subsets of checks.
	profiles?: [string]: #Profile

	// Scenario-based contracts.
	contracts?: [...#Contract & {retries?: <=max_retries}]

	// Performance and resource budgets.
	budgets?: [...#Budget]

	// Audit configuration.
	audit?: #Audit

	// Extension hooks for domain profiles.
	extensions?: [string]: _

	// Monorepo package declarations.
	packages?: [...#Package]
}

// A spec overlay adds contracts beside the base spec for one run
// (maslow verify --overlay). It cannot change anything else.
#Overlay: {
	description?: string
	contracts: [...#Contract]
}

#Project: {
	name:         string & =~"^[a-zA-Z][a-zA-Z0-9_-]*$"
	description?: string
	version?:     string
	license?:     string
	repository?:  string
}

#Toolchain: {
	manager?: "asdf" | "mise" | "nix" | "custom"
	tools?: [...#Tool]
	lockfiles?: [...string]
}

#Tool: {
	name:     string
	version:  string
	required: bool | *true
}

#Ref: {
	kind: "file" | "url" | "package" | "binary" | "container" | "endpoint" | "mcp"
	path: string
	description?: string
	required:     bool | *true

	// MCP server fields.
	if kind == "mcp" {
		name:       string
		source?:    string
		transport?: "stdio" | "sse" | "streamable-http"
		command?:   string
		args?: [...string]
		url?:       string
		env?: [string]: string
	}
}

#Policy: {
	#PolicyRules
	// Rules evaluated and reported as would_* findings without affecting the verdict,
	// so a new rule can be tried on real changes before it is enforced.
	observe?: #PolicyRules
}

// Patterns are repository-relative; a "**" segment matches any depth.
#PolicyRules: {
	// Tracked or unignored files that must not exist.
	deny?: [...string]
	// When set, every changed file must match one of these.
	allow?: [...string]
	// Paths a change must not touch.
	protected?: [...string]
	// Paths a change may touch only with review by their owner.
	gated?: [...#GatedDir]
	// Files a change may remove lines from but never add lines to.
	shrink_only?: [...string]
	// Asks for review when a change deletes tests, adds skips or lowers thresholds.
	test_integrity?: #TestIntegrity
}

#TestIntegrity: {
	// Unset means every built-in language with its defaults. A listed built-in
	// fills unset fields from its defaults; any other language must give files.
	languages?: {
		[#BuiltinTestLanguage]: #TestLanguage
		[!~"^(go|ruby|javascript|python)$"]: #TestLanguage & {files!: [string, ...string]}
	}
	// Files whose numbers are floors, such as coverage minimums.
	thresholds?: [...string]
}

#BuiltinTestLanguage: "go" | "ruby" | "javascript" | "python"

#TestLanguage: {
	// Path patterns of test files.
	files?: [...string]
	// RE2 expressions matching a test declaration; the matched text names the test.
	tests?: [...string]
	// RE2 expressions matching a skip marker.
	skips?: [...string]
}

#GatedDir: {
	path: string
	// Who reviews the change; a GitHub handle or team such as "@org/team".
	owner: string
}

#Checks: {
	runner: [...#CheckRunner]
}

#CheckRunner: {
	name:    string
	kind:    "command" | "script" | "builtin"
	run:     string
	timeout?: string & =~"^[0-9]+(s|m|h)$"
	// Reruns of a failing check; one that fails and then passes is "flaky".
	retries?: #Retries
	tags?: [...string]
}

#Retries: int & >=0

#Profile: {
	description?: string
	checks: [...string]
	// Contract and budget names or tags to run. Omitted runs all; [] runs none.
	contracts?: [...string]
	budgets?: [...string]
	// Refs and policy run on every profile unless turned off here.
	refs?:   bool
	policy?: bool
}

#Contract: {
	name:        string
	description?: string
	tags?: [...string]
	// Environment variables templates may read as {{.env.NAME}}; their values are redacted from evidence.
	env?: [...=~"^[A-Za-z_][A-Za-z0-9_]*$"]
	// Base for step URLs starting with "/"; an endpoint audit target replaces it.
	base_url?: string
	// Reruns of each failing scenario, inside the contract's setup and teardown.
	retries?: #Retries
	setup?: [...#Step]
	teardown?: [...#Step]
	scenarios: [...#Scenario]
}

#Scenario: {
	name: string
	setup?: [...#Step]
	teardown?: [...#Step]
	steps: [...#Step]
	// Repository-relative patterns for files to list in the evidence, such as screenshots.
	artifacts?: [...string]
}

#Step: {
	action: "http" | "cli" | "poll" | "assert" | "capture" | "wait" | "browser"

	// HTTP action fields.
	if action == "http" {
		method:  string
		url:     string
		headers?: [string]: string
		body?:   string
		// Parts of a multipart/form-data body, sent in order instead of body.
		multipart?: [#Part, ...#Part]
		expect?: #Expectation
		matchN(<=1, [{body!: _}, {multipart!: _}])
	}

	// CLI action fields.
	if action == "cli" {
		command: string
		args?: [...string]
		stdin?: string
	}

	// Poll action fields.
	if action == "poll" {
		url:       string
		interval:  string
		timeout:   string
		expect?:   #Expectation
	}

	// Assert action fields.
	if action == "assert" {
		expect: #Expectation
	}

	// Capture action fields.
	if action == "capture" {
		from:  string
		as:    string
	}

	// Wait action fields.
	if action == "wait" {
		duration: string
	}

	// Browser action fields: one headless-browser operation per step, sharing
	// the scenario's cookies. Elements must exist already; only "wait" waits.
	if action == "browser" {
		do: "navigate" | "click" | "fill" | "wait" | "assert"
		// Limit for this operation; for "wait", how long to wait. Defaults to 30s.
		timeout?: string

		if do == "navigate" {
			url: string
		}
		if do == "click" {
			selector: string
		}
		if do == "fill" {
			selector: string
			text:     string
		}
		if do == "wait" {
			selector: string
		}
		if do == "assert" {
			// Without a selector, text_contains reads the whole page.
			selector?: string
			expect:    #BrowserExpectation
		}
	}
}

// One part of a multipart HTTP body: a text field, a fixture file, or inline file content.
#Part: {
	field!:        string & !=""
	value?:        string
	file?:         #FixturePath
	content?:      string
	filename?:     string & !=""
	content_type?: string & !=""

	// Exactly one of value, file or content; a text field has no filename or type.
	matchN(1, [{value!: _}, {file!: _}, {content!: _}])
	matchIf({value!: _}, matchN(0, [{filename!: _}, {content_type!: _}]), _)
	matchIf({content!: _}, {filename!: _}, _)
}

// A slash-separated path relative to where Maslow runs, with no ".", ".." or empty segment.
#FixturePath: string & =~"^[^/\\\\]+(/[^/\\\\]+)*$" & !~"(^|/)\\.\\.?(/|$)"

#BrowserExpectation: {
	text_contains?: string
	url_contains?:  string
	visible?:       bool
	enabled?:       bool
}

#Expectation: {
	status?:   int
	json_path?: string
	value?:    _
	headers?: [string]: string
	body_contains?: string
	body_matches?:  string
	exit_code?: int
	// Multiple JSON path assertions (all must pass).
	assertions?: [...#Assertion]
	// Collection assertions (apply to the array at json_path).
	min_length?: int & >=0
	max_length?: int & >=0
	length?:     int & >=0
	every?:      #Assertion
	some?:       #Assertion
}

#Assertion: {
	json_path: string
	value?:    _
}

#Budget: {
	name: string
	kind: "performance" | "availability" | "rate_limit" | "artifact_size" | "complexity"
	tags?: [...string]

	// Performance budget fields.
	if kind == "performance" {
		target:    string
		p50?:      string
		p90?:      string
		p95?:      string
		p99?:      string
		max?:      string
		error_rate?: number & >=0 & <=1
		concurrency?: int & >0
		duration?:    string
		warmup?:      string
	}

	// Artifact size budget fields.
	if kind == "artifact_size" {
		target: string
		max:    string
	}

	// Availability budget fields.
	if kind == "availability" {
		target:   string
		uptime?:  number & >=0 & <=1
	}

	// Rate limit fields.
	if kind == "rate_limit" {
		target:     string
		max_rps?:   int
		burst?:     int
	}

	// Complexity fields.
	if kind == "complexity" {
		target:    string
		max_lines?: int
		max_functions?: int
		max_depth?:     int
	}
}

#Audit: {
	targets: [...#AuditTarget]
}

// An endpoint target's path is a base URL (it may read {{.env.NAME}}), and its
// checks select contracts by name or tag; omitted runs every contract.
#AuditTarget: {
	kind: "binary" | "container" | "endpoint"
	path: string
	env?: [string]: string
	timeout?: string & =~"^[0-9]+(s|m|h)$"
	retries?: int & >=0
	checks?: [...string]
}

#Package: {
	name: string
	path: string
	checks?: [...string]
	budgets?: [...string]
}
