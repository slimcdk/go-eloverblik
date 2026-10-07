package cmd

import "runtime/debug"

// version is the CLI's version, set at link time by the release build (.goreleaser.yaml):
//
//	-X github.com/slimcdk/go-eloverblik/cmd.version=v{{ .Version }}
//
// It is empty in a binary built any other way, such as go install ...@v1.4.0, and the
// version then comes from the build info the go command embeds in every binary.
var version string

// cliVersion returns the version --version reports: the one set at link time, else the
// main module's version from the build info, which is the tag for go install ...@vX.Y.Z
// and a pseudo-version for a build from a git checkout, else "(devel)".
func cliVersion(linked string, info *debug.BuildInfo, ok bool) string {
	if linked != "" {
		return linked
	}
	if ok && info != nil && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
