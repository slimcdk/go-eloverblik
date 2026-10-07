package cmd

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliVersion(t *testing.T) {
	built := func(version string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/slimcdk/go-eloverblik", Version: version}}
	}

	cases := map[string]struct {
		linked string
		info   *debug.BuildInfo
		ok     bool
		want   string
	}{
		"a release build, versioned at link time": {
			linked: "v1.5.0", info: built("v1.5.0+dirty"), ok: true, want: "v1.5.0",
		},
		"go install of a tagged version": {
			info: built("v1.4.0"), ok: true, want: "v1.4.0",
		},
		"a build from a git checkout": {
			info: built("v1.4.1-0.20261008120000-0123456789ab+dirty"), ok: true,
			want: "v1.4.1-0.20261008120000-0123456789ab+dirty",
		},
		"a build without a version in its build info": {
			info: built(""), ok: true, want: "(devel)",
		},
		"a binary without build info": {
			want: "(devel)",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, cliVersion(tc.linked, tc.info, tc.ok))
		})
	}
}

// TestVersionFlag covers --version and -v on the root command: they print the CLI's
// version and nothing else, without a token.
func TestVersionFlag(t *testing.T) {
	require.NotEmpty(t, rootCmd.Version)

	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			out, err := execute(t, flag)

			require.NoError(t, err)
			assert.Equal(t, "go-eloverblik version "+rootCmd.Version, out)
		})
	}
}
