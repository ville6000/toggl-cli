package cmd

import "runtime/debug"

// version is the release being built, stamped by the release build with
// -ldflags "-X github.com/ville6000/toggl-cli/cmd.version=<version>".
var version string

// buildVersion returns the version --version reports: the stamped release
// version, else the module version Go recorded in the binary (the tag for
// `go install ...@vX.Y.Z`, a git-derived pseudo-version for `go build` in a
// checkout), else "dev".
func buildVersion() string {
	if version != "" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}
