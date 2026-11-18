package verify

import (
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func overlayOf(path string, steps ...spec.Step) *spec.LoadedOverlay {
	return &spec.LoadedOverlay{
		Path:    path,
		SHA256:  "abc",
		Overlay: &spec.Overlay{Contracts: []spec.Contract{{Name: "acceptance", Scenarios: []spec.Scenario{{Name: "s", Steps: steps}}}}},
	}
}

// A profile that selects no contracts must still run the overlay's, or a
// change's acceptance scenarios could be skipped by picking a profile.
func TestRun_OverlayContractsRunOnEveryProfile(t *testing.T) {
	mas := &spec.MAS{
		Contracts: []spec.Contract{{Name: "smoke", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{{Action: "cli", Command: "true"}}}}}},
		Profiles:  map[string]spec.Profile{"quick": {Checks: []string{}, Contracts: []string{}}},
	}
	overlay := overlayOf("change/acceptance.yaml", spec.Step{Action: "cli", Command: "false"}, spec.Step{Action: "assert", Expect: &spec.Expectation{ExitCode: new(int)}})

	report, err := Run(mas, Options{Profile: "quick", Overlays: []*spec.LoadedOverlay{overlay}})
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Contracts) != 1 {
		t.Fatalf("expected only the overlay's contract, got %+v", report.Contracts)
	}
	got := report.Contracts[0]
	if got.Name != "acceptance/s" || got.Overlay != "change/acceptance.yaml" || got.Status != "fail" {
		t.Errorf("unexpected overlay result: %+v", got)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected a failing overlay to fail the run, got %s", report.Verdict)
	}
	if len(report.Overlays) != 1 || report.Overlays[0].SHA256 != "abc" {
		t.Errorf("expected the overlay recorded with its digest, got %+v", report.Overlays)
	}
}

func TestRun_OverlayCannotReplaceSpecContract(t *testing.T) {
	mas := &spec.MAS{
		Contracts: []spec.Contract{{Name: "acceptance"}},
		Profiles:  map[string]spec.Profile{"quick": {}},
	}
	if _, err := Run(mas, Options{Profile: "quick", Overlays: []*spec.LoadedOverlay{overlayOf("o.yaml")}}); err == nil {
		t.Error("expected a name collision to stop the run")
	}
}
