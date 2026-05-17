package version

import (
	"fmt"
	"runtime/debug"
)

// Value is the base version string. Overridden at build time via ldflags when a
// git tag exists; falls back to this default for untagged dev builds.
var Value = "v1.0.0"

// tagged is set to "true" via ldflags on tagged release builds. When set,
// String() returns Value as-is without appending a commit suffix, so that
// release binaries never show a false "-dirty" marker from build artifacts.
var tagged = ""

// String returns the version. On tagged release builds it returns Value exactly.
// On dev builds it appends a short commit hash and dirty flag from build metadata.
func String() string {
	if tagged == "true" {
		return Value
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Value
	}
	return formatVersion(Value, info.Settings)
}

// formatVersion applies VCS settings to base and returns the formatted version string.
// It is a pure function extracted for testability.
func formatVersion(base string, settings []debug.BuildSetting) string {
	var commit string
	var dirty bool
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			n := min(7, len(s.Value))
			commit = s.Value[:n]
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if commit == "" {
		return base
	}

	suffix := commit
	if dirty {
		suffix += "-dirty"
	}
	return fmt.Sprintf("%s (%s)", base, suffix)
}
