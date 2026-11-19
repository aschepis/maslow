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
	result.FailedStep = redactStep(result.FailedStep, r)
	if result.Attempts != nil {
		attempts := make([]evidence.ContractAttempt, len(result.Attempts))
		for i, a := range result.Attempts {
			a.Error = r.Replace(a.Error)
			a.Output = r.Replace(a.Output)
			a.FailedStep = redactStep(a.FailedStep, r)
			attempts[i] = a
		}
		result.Attempts = attempts
	}
	return result
}

func redactStep(failed *evidence.FailedStep, r *strings.Replacer) *evidence.FailedStep {
	if failed == nil {
		return nil
	}
	step := *failed
	step.Target = r.Replace(step.Target)
	step.PageURL = r.Replace(step.PageURL)
	if step.Console != nil {
		console := make([]string, len(step.Console))
		for i, m := range step.Console {
			console[i] = r.Replace(m)
		}
		step.Console = console
	}
	return &step
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
