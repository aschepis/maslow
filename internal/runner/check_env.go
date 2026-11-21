package runner

import (
	"os"
	"slices"
)

// baseCheckEnv is what every check gets without listing it: where to find tools, home, temp and
// cache directories, locale and the CI flag. None of these normally carry a secret (ADR 0046).
var baseCheckEnv = []string{
	"PATH", "HOME", "TMPDIR", "USER", "LOGNAME",
	"LANG", "LC_ALL", "LC_CTYPE",
	"XDG_CACHE_HOME", "XDG_CONFIG_HOME",
	"CI",
	// Windows programs, and Go's cache lookup there, fail without these.
	"SystemRoot", "ComSpec", "PATHEXT", "TEMP", "TMP", "USERPROFILE", "LOCALAPPDATA", "APPDATA",
}

// checkEnv returns the environment for a check: the set variables among the
// base set and the check's env list, in NAME=value form.
func checkEnv(listed []string) []string {
	names := slices.Concat(baseCheckEnv, listed)
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
