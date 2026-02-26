package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// execute is a helper function to capture the output of a cobra command.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

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
