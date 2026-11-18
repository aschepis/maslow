package buildinfo

import "testing"

func TestVersion_StampedValueWinsAndDropsV(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })

	version = "v0.6.0"
	if got := Version(); got != "0.6.0" {
		t.Errorf("Version() = %q, want 0.6.0", got)
	}

	version = "0.6.0-rc.1"
	if got := Version(); got != "0.6.0-rc.1" {
		t.Errorf("Version() = %q, want 0.6.0-rc.1", got)
	}
}

func TestVersion_NeverEmpty(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	version = ""
	if Version() == "" {
		t.Error("expected a non-empty fallback version")
	}
}
