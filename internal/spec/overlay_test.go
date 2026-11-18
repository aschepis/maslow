package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOverlay(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "overlay.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const acceptanceOverlay = `description: Acceptance scenarios for one change
contracts:
  - name: acceptance
    scenarios:
      - name: echoes
        steps:
          - action: cli
            command: echo
            args: [hi]
`

func TestReadOverlay(t *testing.T) {
	loaded, err := ReadOverlay(writeOverlay(t, acceptanceOverlay))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Overlay.Contracts) != 1 || loaded.Overlay.Contracts[0].Name != "acceptance" {
		t.Errorf("unexpected contracts: %+v", loaded.Overlay.Contracts)
	}
	if len(loaded.SHA256) != 64 {
		t.Errorf("expected a sha256 hex digest, got %q", loaded.SHA256)
	}
}

func TestReadOverlay_RejectsAnythingButContracts(t *testing.T) {
	for name, body := range map[string]string{
		"policy":    acceptanceOverlay + "policy:\n  deny: []\n",
		"profiles":  acceptanceOverlay + "profiles:\n  quick:\n    checks: []\n",
		"bad step":  "contracts:\n  - name: c\n    scenarios:\n      - name: s\n        steps:\n          - action: teleport\n",
		"no list":   "description: empty\n",
		"not  yaml": "contracts: [",
	} {
		if _, err := ReadOverlay(writeOverlay(t, body)); err == nil {
			t.Errorf("%s: expected the overlay to be rejected", name)
		}
	}
}

func TestCheckOverlayNames(t *testing.T) {
	a, _ := ReadOverlay(writeOverlay(t, acceptanceOverlay))
	b, _ := ReadOverlay(writeOverlay(t, acceptanceOverlay))

	if err := CheckOverlayNames(&MAS{Contracts: []Contract{{Name: "smoke"}}}, []*LoadedOverlay{a}); err != nil {
		t.Errorf("expected distinct names to pass, got %v", err)
	}

	err := CheckOverlayNames(&MAS{Contracts: []Contract{{Name: "acceptance"}}}, []*LoadedOverlay{a})
	if err == nil || !strings.Contains(err.Error(), "already defined by the spec") {
		t.Errorf("expected an overlay to be unable to replace a spec contract, got %v", err)
	}

	err = CheckOverlayNames(&MAS{}, []*LoadedOverlay{a, b})
	if err == nil || !strings.Contains(err.Error(), "already defined by overlay") {
		t.Errorf("expected two overlays to be unable to share a name, got %v", err)
	}
}
