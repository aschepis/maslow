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

	// Check runners.
	checks?: #Checks

	// Profiles select subsets of checks.
	profiles?: [string]: #Profile

	// Scenario-based contracts.
	contracts?: [...#Contract]

	// Performance and resource budgets.
	budgets?: [...#Budget]

	// Audit configuration.
	audit?: #Audit

	// Extension hooks for domain profiles.
	extensions?: [string]: _

	// Monorepo package declarations.
	packages?: [...#Package]
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
	kind: "file" | "url" | "package" | "binary" | "container" | "endpoint"
	path: string
	description?: string
	required:     bool | *true
}

#Policy: {
	deny?: [...string]
	allow?: [...string]
	protected?: [...string]
	gated?: [...#GatedDir]
}

#GatedDir: {
	path:  string
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
	tags?: [...string]
}

#Profile: {
	description?: string
	checks: [...string]
}

#Contract: {
	name:        string
	description?: string
	scenarios: [...#Scenario]
}

#Scenario: {
	name: string
	steps: [...#Step]
}

#Step: {
	action: "http" | "cli" | "poll" | "assert" | "capture" | "wait"

	// HTTP action fields.
	if action == "http" {
		method:  string
		url:     string
		headers?: [string]: string
		body?:   string
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
}

#Expectation: {
	status?:   int
	json_path?: string
	value?:    _
	headers?: [string]: string
	body_contains?: string
	exit_code?: int
}

#Budget: {
	name: string
	kind: "performance" | "availability" | "rate_limit" | "artifact_size" | "complexity"

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
