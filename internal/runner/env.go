package runner

import (
	"os"
	"slices"
)

// baseEnv is what every check, CLI step and audit binary gets without listing it: where to find
// tools, home, temp and cache directories, locale and the CI flag. None normally carry a secret (ADR 0046).
var baseEnv = []string{
	"PATH", "HOME", "TMPDIR", "USER", "LOGNAME",
	"LANG", "LC_ALL", "LC_CTYPE",
	"XDG_CACHE_HOME", "XDG_CONFIG_HOME",
	"CI",
	// Windows programs, and Go's cache lookup there, fail without these.
	"SystemRoot", "ComSpec", "PATHEXT", "TEMP", "TMP", "USERPROFILE", "LOCALAPPDATA", "APPDATA",
}

// ScrubbedEnv returns the set variables among the base set and listed, in NAME=value form,
// for a process that must not see the rest of Maslow's environment (ADRs 0046 and 0048).
func ScrubbedEnv(listed []string) []string {
	names := slices.Concat(baseEnv, listed)
	slices.Sort(names)
	names = slices.Compact(names)

	// Empty rather than nil: exec gives a nil Env the whole parent environment.
	env := []string{}
	for _, name := range names {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	return env
}
