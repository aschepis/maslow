// Package buildinfo holds the version stamped into the binary, so that
// `maslow version` and the evidence file always agree.
package buildinfo

import "runtime/debug"

// Set via -ldflags "-X github.com/aschepis/maslow/internal/buildinfo.version=...".
var (
	version = ""
	gitSHA  = ""
)

// Version returns the stamped version, the module version for a binary
// built by `go install module@version`, or "dev".
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// GitSHA returns the commit the binary was built from, or "unknown".
func GitSHA() string {
	if gitSHA != "" {
		return gitSHA
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}
	return "unknown"
}
