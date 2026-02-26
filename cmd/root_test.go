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

// execute is a helper function to capture the output of a cobra command.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset flag state to avoid pollution between sequential calls
	resetCommandFlags(rootCmd)

	// Redirect stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a buffer to capture the output
	var buf bytes.Buffer

	// Execute the command
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	// Stop redirecting stdout
	w.Close()
	os.Stdout = old

	// Read the output from the pipe
	_, readErr := io.Copy(&buf, r)
	assert.NoError(t, readErr)

	return strings.TrimSpace(buf.String()), err
}

func TestRootCmd(t *testing.T) {
	// Test --help flag
	out, err := execute(t, "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "A CLI for the Danish Eloverblik platform")

	// Test with no arguments
	_, err = execute(t)
	assert.NoError(t, err)

	// Test with an invalid command
	_, err = execute(t, "invalid-command")
	assert.Error(t, err)
}

func TestExecute(t *testing.T) {
	// Reset state from previous tests
	resetCommandFlags(rootCmd)
	rootCmd.SetArgs([]string{})

	// Redirect stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a buffer to capture the output
	var buf bytes.Buffer

	// Execute the command
	Execute()

	// Stop redirecting stdout
	w.Close()
	os.Stdout = old

	// Read the output from the pipe
	_, readErr := io.Copy(&buf, r)
	assert.NoError(t, readErr)
}
