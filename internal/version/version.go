package version

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"
)

const programName = "nobl9-language-server"

// BuildVersion defaults to VERSION file contents.
// This is necessary since we don't have control over build flags when installed through `go install`.
//
//go:embed VERSION
var embeddedBuildVersion string

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
		version = embeddedBuildVersion
	}
	return strings.TrimSpace(version)
}
