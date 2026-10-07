package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetCommandFlags resets the "Changed" state and values of all flags in the
// command tree so that sequential execute() calls in tests don't pollute each other.
func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		if f.Value != nil {
			_ = f.Value.Set(f.DefValue)
		}
	})
	for _, c := range cmd.Commands() {
		resetCommandFlags(c)
	}
}

// captureStdout returns what run writes to os.Stdout. The pipe is drained while run is
// still writing: a pipe holds only a few KB (about 4 KB on Windows), so reading it after
// run returns would block a command with more output than that, such as a completion
// script, forever.
func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	var buf bytes.Buffer
	copied := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, r)
		copied <- err
	}()

	old := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = old
		_ = r.Close()
	}()

	run()

	require.NoError(t, w.Close())
	require.NoError(t, <-copied)

	return buf.String()
}

// execute is a helper function to capture the output of a cobra command.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset flag state to avoid pollution between sequential calls
	resetCommandFlags(rootCmd)

	var err error
	out := captureStdout(t, func() {
		rootCmd.SetArgs(args)
		err = rootCmd.Execute()
	})

	return strings.TrimSpace(out), err
}

func TestRootCmd(t *testing.T) {
	// Test --help flag
	out, err := execute(t, "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "A CLI for the Danish Eloverblik platform")

	// Test with no arguments
	_, err = execute(t)
	require.NoError(t, err)

	// Test with an invalid command
	_, err = execute(t, "invalid-command")
	assert.Error(t, err)
}

func TestPrintResponseHeadersFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("print-response-headers")
	assert.NotNil(t, flag, "rootCmd should have a persistent --print-response-headers flag")
	assert.Equal(t, "false", flag.DefValue)

	printHeaders, err := rootCmd.PersistentFlags().GetBool("print-response-headers")
	require.NoError(t, err)
	assert.False(t, printHeaders)
}

func TestExecute(t *testing.T) {
	// Reset state from previous tests
	resetCommandFlags(rootCmd)
	rootCmd.SetArgs([]string{})

	out := captureStdout(t, Execute)
	assert.Contains(t, out, "A CLI for the Danish Eloverblik platform")
}
