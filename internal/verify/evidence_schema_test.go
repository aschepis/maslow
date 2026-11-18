package verify

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/jsonschema"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// Real output from a run that exercises every section must satisfy the
// checked-in JSON Schema, so the published format cannot drift silently.
func TestEvidenceMatchesSchema(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "maslow.yaml", gatedSpec)
	write(t, dir, "locked.txt", "v1")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	write(t, dir, "locked.txt", "v2")
	write(t, dir, ".env", "SECRET=1")
	t.Chdir(dir)

	exitZero := 0
	loaded, err := spec.Read("maslow.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	mas := loaded.MAS
	mas.Checks.Runner = append(mas.Checks.Runner, spec.CheckRunner{Name: "noisy", Kind: "command", Run: "seq 1 5000"})
	mas.Profiles["full"] = spec.Profile{Checks: []string{"gate", "ok", "noisy"}}
	mas.Contracts = []spec.Contract{{Name: "c", Scenarios: []spec.Scenario{
		{
			Name: "s",
			Steps: []spec.Step{
				{Action: "cli", Command: "true"},
				{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
			},
		},
		{
			Name: "failing",
			Steps: []spec.Step{
				{Action: "cli", Command: "seq", Args: []string{"1", "5000"}},
				{Action: "assert", Expect: &spec.Expectation{BodyContains: "absent"}},
			},
			Artifacts: []string{"locked.txt"},
		},
	}}}
	mas.Budgets = []spec.Budget{{Name: "size", Kind: "artifact_size", Target: "maslow.yaml", Max: "1MB"}}
	mas.Refs = []spec.Ref{{Kind: "file", Path: "maslow.yaml", Required: true}}
	mas.Policy = &spec.Policy{
		Deny:      []string{"**/.env"},
		Protected: []string{"locked.txt"},
		Gated:     []spec.GatedDir{{Path: "infra/**", Owner: "ops"}},
	}

	head := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))
	overlay := &spec.LoadedOverlay{Path: "acceptance.yaml", SHA256: "0123", Overlay: &spec.Overlay{Contracts: []spec.Contract{{
		Name:      "acceptance",
		Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{{Action: "cli", Command: "true"}}}},
	}}}}
	report, err := Run(mas, Options{Profile: "full", BaseSHA: head, SpecPath: "maslow.yaml", SpecSHA256: loaded.SHA256, SpecFromBase: true, Overlays: []*spec.LoadedOverlay{overlay}})
	if err != nil {
		t.Fatal(err)
	}
	report.CapOutput(evidence.DefaultOutputLimit)

	path := filepath.Join(t.TempDir(), "verify.json")
	if err := report.Write(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	validateAgainstSchema(t, data)

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"contracts", "budgets", "refs", "policy", "base_sha", "git_sha", "overlays"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("expected the fixture to exercise %q", key)
		}
	}

	mutations := map[string]func(m map[string]any){
		"unknown field": func(m map[string]any) { m["surprise"] = true },
		"bad verdict":   func(m map[string]any) { m["verdict"] = "maybe" },
	}
	for name, mutate := range mutations {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		mutate(m)
		bad, _ := json.Marshal(m)
		ctx := cuecontext.New()
		if err := compileEvidenceSchema(t, ctx).Unify(ctx.CompileBytes(bad)).Validate(cue.Concrete(true)); err == nil {
			t.Errorf("expected the schema to reject evidence with %s", name)
		}
	}
}

func TestSpecLoadFailureMatchesSchema(t *testing.T) {
	report := SpecLoadFailure(Options{Profile: "ci", SpecPath: "maslow.yaml", BaseSHA: "abc"}, errors.New("maslow.yaml at abc is invalid"))
	if report.Verdict != "fail" {
		t.Errorf("expected fail, got %s", report.Verdict)
	}
	path := filepath.Join(t.TempDir(), "verify.json")
	if err := report.Write(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	validateAgainstSchema(t, data)
}

func TestEvidenceSchemaRejectsUnknownFields(t *testing.T) {
	ctx := cuecontext.New()
	schemaVal := compileEvidenceSchema(t, ctx)
	bad := ctx.CompileString(`{"evidence_version": "2", "surprise": true}`)
	if err := schemaVal.Unify(bad).Validate(cue.Concrete(true)); err == nil {
		t.Error("expected the schema to reject unknown and missing fields")
	}
}

func validateAgainstSchema(t *testing.T, data []byte) {
	t.Helper()
	ctx := cuecontext.New()
	schemaVal := compileEvidenceSchema(t, ctx)
	dataVal := ctx.CompileBytes(data)
	if err := dataVal.Err(); err != nil {
		t.Fatalf("evidence is not valid JSON: %v", err)
	}
	if err := schemaVal.Unify(dataVal).Validate(cue.Concrete(true)); err != nil {
		t.Fatalf("evidence does not match schema/evidence.schema.json: %v\n%s", err, data)
	}
}

func compileEvidenceSchema(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "schema", "evidence.schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	file, err := jsonschema.Extract(ctx.CompileBytes(raw), &jsonschema.Config{})
	if err != nil {
		t.Fatalf("cannot read JSON Schema: %v", err)
	}
	v := ctx.BuildFile(file)
	if err := v.Err(); err != nil {
		t.Fatalf("cannot build JSON Schema: %v", err)
	}
	return v
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(out)
}
