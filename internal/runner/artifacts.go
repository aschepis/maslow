package runner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
)

// attachArtifacts adds the files matching patterns to result's artifacts. A
// bad pattern is a spec mistake, so it turns a passing result into an error.
func attachArtifacts(result *evidence.ContractResult, patterns []string) {
	files, err := collectArtifacts(patterns)
	for _, f := range result.Artifacts {
		if !slices.Contains(files, f) {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	result.Artifacts = files
	if err == nil {
		return
	}
	msg := fmt.Sprintf("artifacts: %v", err)
	if result.Status == "pass" {
		result.Status = "error"
		result.Error = msg
		return
	}
	result.Error += "; " + msg
}

// collectArtifacts returns the existing files matching any of the
// repository-relative patterns, sorted and without duplicates.
func collectArtifacts(patterns []string) ([]string, error) {
	seen := map[string]bool{}
	for _, pattern := range patterns {
		if err := validatePattern(pattern); err != nil {
			return nil, err
		}
		root := literalPrefix(pattern)
		err := filepath.WalkDir(filepath.FromSlash(root), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel := filepath.ToSlash(p)
			if ok, _ := matchPath(pattern, rel); ok {
				seen[rel] = true
			}
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// literalPrefix is the directory a pattern's matches must live under, so a
// walk need not cover the whole repository.
func literalPrefix(pattern string) string {
	segs := strings.Split(pattern, "/")
	var literal []string
	for _, seg := range segs[:len(segs)-1] {
		if strings.ContainsAny(seg, `*?[\`) {
			break
		}
		literal = append(literal, seg)
	}
	if len(literal) == 0 {
		return "."
	}
	return path.Join(literal...)
}
