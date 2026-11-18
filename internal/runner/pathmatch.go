package runner

import (
	"path"
	"strings"
)

// matchPath reports whether a slash-separated, repository-relative path
// matches a policy pattern. A "**" segment matches zero or more whole
// segments; every other segment follows path.Match.
func matchPath(pattern, name string) (bool, error) {
	if err := validatePattern(pattern); err != nil {
		return false, err
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func validatePattern(pattern string) error {
	for _, seg := range strings.Split(pattern, "/") {
		if _, err := path.Match(seg, ""); err != nil {
			return err
		}
	}
	return nil
}

func matchSegments(pattern, name []string) (bool, error) {
	if len(pattern) == 0 {
		return len(name) == 0, nil
	}

	if pattern[0] == "**" {
		for skip := 0; skip <= len(name); skip++ {
			ok, err := matchSegments(pattern[1:], name[skip:])
			if ok || err != nil {
				return ok, err
			}
		}
		return false, nil
	}

	if len(name) == 0 {
		return false, nil
	}
	ok, err := path.Match(pattern[0], name[0])
	if !ok || err != nil {
		return false, err
	}
	return matchSegments(pattern[1:], name[1:])
}

// matchingFiles returns the files that match pattern, preserving input order.
func matchingFiles(pattern string, files []string) ([]string, error) {
	var matched []string
	for _, f := range files {
		ok, err := matchPath(pattern, f)
		if err != nil {
			return nil, err
		}
		if ok {
			matched = append(matched, f)
		}
	}
	return matched, nil
}
