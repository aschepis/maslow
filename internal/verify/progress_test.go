package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/progress"
	"github.com/aschepis/maslow/internal/spec"
)

func progressSpec(t *testing.T) *spec.MAS {
	// Fails the first time it runs and passes after, so one retry makes it flaky.
	script := filepath.Join(t.TempDir(), "wobbly.sh")
	if err := os.WriteFile(script, []byte(`[ -f "$0.ran" ] && exit 0; touch "$0.ran"; exit 1`), 0o755); err != nil {
		t.Fatal(err)
	}
	exitZero := 0
	return &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{
				{Name: "ok", Kind: "command", Run: "true"},
				{Name: "broken", Kind: "command", Run: "false"},
				{Name: "wobbly", Kind: "command", Run: "sh " + script, Retries: 1},
			},
		},
		Profiles: map[string]spec.Profile{
			"quick": {Checks: []string{"ok", "broken", "wobbly"}},
		},
		Contracts: []spec.Contract{{
			Name: "cli",
			Scenarios: []spec.Scenario{
				{Name: "a", Steps: []spec.Step{
					{Action: "cli", Command: "true"},
					{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
				}},
				{Name: "b", Steps: []spec.Step{
					{Action: "cli", Command: "false"},
					{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
				}},
			},
		}},
		Refs: []spec.Ref{{Kind: "file", Path: "progress_test.go"}},
	}
}

type recorded struct {
	Phase    progress.Phase
	Total    int
	Kind     string
	Name     string
	Statuses []string
	Attempts int
}

func TestRun_EmitsProgressInOrder(t *testing.T) {
	var got []recorded
	opts := Options{Profile: "quick", Progress: func(e progress.Event) {
		got = append(got, recorded{e.Phase, e.Total, e.Kind, e.Name, e.Statuses, e.Attempts})
	}}
	if _, err := Run(progressSpec(t), opts); err != nil {
		t.Fatal(err)
	}

	if len(got) == 0 {
		t.Fatal("no progress events")
	}
	policy := got[len(got)-1]
	if policy.Phase != progress.Finish || policy.Kind != "policy" {
		t.Errorf("last event = %+v, want policy finish", policy)
	}
	got = got[:len(got)-2]

	want := []recorded{
		{Phase: progress.Plan, Total: 6},
		{Phase: progress.Start, Kind: "check", Name: "ok"},
		{Phase: progress.Finish, Kind: "check", Name: "ok", Statuses: []string{"pass"}, Attempts: 1},
		{Phase: progress.Start, Kind: "check", Name: "broken"},
		{Phase: progress.Finish, Kind: "check", Name: "broken", Statuses: []string{"fail"}, Attempts: 1},
		{Phase: progress.Start, Kind: "check", Name: "wobbly"},
		{Phase: progress.Finish, Kind: "check", Name: "wobbly", Statuses: []string{evidence.StatusFlaky}, Attempts: 2},
		{Phase: progress.Start, Kind: "contract", Name: "cli"},
		{Phase: progress.Finish, Kind: "contract", Name: "cli", Statuses: []string{"pass", "fail"}, Attempts: 1},
		{Phase: progress.Start, Kind: "ref", Name: "progress_test.go"},
		{Phase: progress.Finish, Kind: "ref", Name: "progress_test.go", Statuses: []string{"pass"}, Attempts: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events:\n got %+v\nwant %+v", got, want)
	}
}

func TestRun_ProgressLeavesEvidenceUnchanged(t *testing.T) {
	without, err := Run(progressSpec(t), Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	with, err := Run(progressSpec(t), Options{Profile: "quick", Progress: func(progress.Event) {}})
	if err != nil {
		t.Fatal(err)
	}
	if a, b := normalizedJSON(t, without), normalizedJSON(t, with); a != b {
		t.Errorf("evidence differs with progress:\n%s\n%s", a, b)
	}
}

// normalizedJSON zeroes the fields that differ between any two runs.
func normalizedJSON(t *testing.T, r *evidence.Report) string {
	t.Helper()
	r.Timestamp = ""
	for i := range r.CheckResults {
		r.CheckResults[i].Duration = 0
		for j := range r.CheckResults[i].Attempts {
			r.CheckResults[i].Attempts[j].Duration = 0
		}
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
