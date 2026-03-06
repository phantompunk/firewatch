package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Version returns the version and short commit hash of the current build.
// Falls back to "dev" and "unknown" for local builds.
func Version() (version, commit string) {
	version, commit = "dev", "unknown"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	fullVersion := info.Main.Version
	if fullVersion != "" && fullVersion != "(devel)" && !strings.Contains(fullVersion, "-") {
		version = fullVersion
	} else if parts := strings.SplitN(fullVersion, "-", 2); parts[0] != "" && parts[0] != "(devel)" {
		version = parts[0]
	}

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			commit = s.Value
			if len(commit) > 7 {
				commit = commit[:7]
			}
			break
		}
	}

	return
}
