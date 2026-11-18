package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/schema"
)

// ErrNotAtCommit means the spec file does not exist at the requested commit.
var ErrNotAtCommit = errors.New("spec does not exist at that commit")

// Loaded is a validated spec together with where it came from.
type Loaded struct {
	MAS    *MAS
	Path   string
	SHA256 string
	// Commit is the commit the spec was read from, or "" for the working tree.
	Commit string
}

// Read reads, schema-validates and parses the spec at path. With a non-empty
// atCommit the spec is read from that commit instead of the working tree.
func Read(path, atCommit string) (*Loaded, error) {
	data, source, err := readBytes(path, atCommit)
	if err != nil {
		return nil, err
	}

	result, err := schema.Validate(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	if !result.Valid {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("%s is invalid:\n  %s", source, strings.Join(msgs, "\n  "))
	}

	mas, err := Parse(data, source)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)
	return &Loaded{MAS: mas, Path: path, SHA256: hex.EncodeToString(sum[:]), Commit: atCommit}, nil
}

func readBytes(path, atCommit string) ([]byte, string, error) {
	if atCommit == "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, path, fmt.Errorf("failed to read %s: %w", path, err)
		}
		return data, path, nil
	}

	if filepath.IsAbs(path) {
		return nil, path, fmt.Errorf("--spec must be a relative path when reading the spec from a commit, got %s", path)
	}
	source := fmt.Sprintf("%s at %s", path, shortSHA(atCommit))
	rel := "./" + filepath.ToSlash(path)
	exists, err := gitrepo.ExistsAt("", atCommit, rel)
	if err != nil {
		return nil, source, fmt.Errorf("failed to read %s: %w", source, err)
	}
	if !exists {
		return nil, source, fmt.Errorf("%s: %w", source, ErrNotAtCommit)
	}
	data, err := gitrepo.Show("", atCommit, rel)
	if err != nil {
		return nil, source, fmt.Errorf("failed to read %s: %w", source, err)
	}
	return data, source, nil
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
