package version

import (
	_ "embed"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const programName = "nobl9-language-server"

// Set during build time.
var (
	BuildGitRevision string
	BuildGitBranch   string
	BuildVersion     string
)

func GetUserAgent() string {
	return fmt.Sprintf("%s/%s-%s-%s (%s %s %s)",
		programName, GetVersion(), BuildGitBranch, BuildGitRevision,
		runtime.GOOS, runtime.GOARCH, runtime.Version(),
	)
}

func GetVersion() string {
	version := BuildVersion
	if version == "" {
		version = getRuntimeVersion()
	}
	return strings.TrimSpace(version)
}

func getRuntimeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "0.0.0"
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}
