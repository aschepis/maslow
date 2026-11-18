package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/aschepis/maslow/internal/schema"
)

// Overlay holds contracts that run beside the base spec for one run.
type Overlay struct {
	Description string     `yaml:"description,omitempty"`
	Contracts   []Contract `yaml:"contracts"`
}

// LoadedOverlay is a validated overlay together with where it came from.
type LoadedOverlay struct {
	Overlay *Overlay
	Path    string
	SHA256  string
}

// ReadOverlay reads, schema-validates and parses the overlay at path.
func ReadOverlay(path string) (*LoadedOverlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read overlay %s: %w", path, err)
	}

	result, err := schema.ValidateOverlay(data)
	if err != nil {
		return nil, fmt.Errorf("overlay %s: %w", path, err)
	}
	if !result.Valid {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("overlay %s is invalid (an overlay may hold only description and contracts):\n  %s", path, strings.Join(msgs, "\n  "))
	}

	var overlay Overlay
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return nil, fmt.Errorf("failed to parse overlay %s: %w", path, err)
	}
	if len(overlay.Contracts) == 0 {
		return nil, fmt.Errorf("overlay %s has no contracts", path)
	}

	sum := sha256.Sum256(data)
	return &LoadedOverlay{Overlay: &overlay, Path: path, SHA256: hex.EncodeToString(sum[:])}, nil
}

// CheckOverlayNames fails when an overlay contract shares a name with a spec
// contract or another overlay's, so an overlay can add scenarios but never
// replace or shadow the ones the spec already enforces.
func CheckOverlayNames(mas *MAS, overlays []*LoadedOverlay) error {
	owner := map[string]string{}
	for _, c := range mas.Contracts {
		owner[c.Name] = "the spec"
	}
	for _, o := range overlays {
		for _, c := range o.Overlay.Contracts {
			if prev, ok := owner[c.Name]; ok {
				return fmt.Errorf("overlay %s: contracts.%s is already defined by %s", o.Path, c.Name, prev)
			}
			owner[c.Name] = "overlay " + o.Path
		}
	}
	return nil
}
