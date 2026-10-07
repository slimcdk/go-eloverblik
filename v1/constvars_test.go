package eloverblik

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPackageEmbedsTheTimeZoneDatabase guards the time/tzdata import. A release binary is
// built with -trimpath and runs where Go is not installed, and Windows has no time zone
// database of its own, so Europe/Copenhagen loads there only from the copy the package
// links. A test binary built by go test still finds the toolchain's zoneinfo.zip, so
// loading the zone here would pass with or without the import; the go command is asked
// what the package links instead.
func TestPackageEmbedsTheTimeZoneDatabase(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("the go command is not on PATH")
	}
	require.NoError(t, err)
	require.True(t, slices.Contains(strings.Fields(string(out)), "time/tzdata"),
		"package eloverblik does not import time/tzdata")
}

func TestMustLoadLocationPanicsOnAZoneItCannotLoad(t *testing.T) {
	require.PanicsWithError(t, "eloverblik: load time zone Nowhere/Nothing: unknown time zone Nowhere/Nothing", func() {
		mustLoadLocation("Nowhere/Nothing")
	})
}
