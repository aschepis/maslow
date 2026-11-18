package runner

import (
	"os"
	"sort"
	"strings"

	"github.com/aschepis/maslow/internal/evidence"
)

// minRedactLen keeps short values such as "1" or "dev" from mangling evidence.
const minRedactLen = 4

// allowedEnv returns the set variables among names. Unset names are left out,
// so a template that reads one fails instead of substituting an empty string.
func allowedEnv(names []string) map[string]string {
	env := make(map[string]string, len(names))
	for _, name := range names {
		if v, ok := os.LookupEnv(name); ok {
			env[name] = v
		}
	}
	return env
}

// redact replaces every allow-listed value in the parts of a result that
// carry text from the run, so evidence can be stored without leaking secrets.
func redact(result evidence.ContractResult, env map[string]string) evidence.ContractResult {
	r := redactor(env)
	if r == nil {
		return result
	}
	result.Error = r.Replace(result.Error)
	result.Output = r.Replace(result.Output)
	if result.FailedStep != nil {
		step := *result.FailedStep
		step.Target = r.Replace(step.Target)
		result.FailedStep = &step
	}
	return result
}

// redactor replaces longer values first, so a value containing another is
// redacted whole.
func redactor(env map[string]string) *strings.Replacer {
	names := make([]string, 0, len(env))
	for name, v := range env {
		if len(v) >= minRedactLen {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Slice(names, func(i, j int) bool {
		if len(env[names[i]]) != len(env[names[j]]) {
			return len(env[names[i]]) > len(env[names[j]])
		}
		return names[i] < names[j]
	})

	var pairs []string
	for _, name := range names {
		pairs = append(pairs, env[name], "[redacted:"+name+"]")
	}
	return strings.NewReplacer(pairs...)
}
