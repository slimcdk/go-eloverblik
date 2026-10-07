package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
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

// TestRootHelpGroupsCommands checks that the root help lists each group's commands under
// the group's name, so every command the CLI has can be seen at once.
func TestRootHelpGroupsCommands(t *testing.T) {
	out, err := execute(t, "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "\n  customer\n    add-relation ")
	assert.Contains(t, out, "\n  thirdparty\n    alive ")
	assert.Contains(t, out, "\n    timeseries ")
	assert.Contains(t, out, "--token")
}

// TestSubcommandHelp checks that every command below the root gets cobra's own help: the
// root's grouped listing has no room for how to call a command, so it showed a
// "[command]" usage line for a command that takes arguments and left out the Long text,
// the examples and the global flags, --token among them.
func TestSubcommandHelp(t *testing.T) {
	t.Run("a leaf command shows its arguments, its flags and the global flags", func(t *testing.T) {
		out, err := execute(t, "customer", "timeseries", "--help")
		require.NoError(t, err)
		assert.Contains(t, out, "go-eloverblik customer timeseries <metering-id> [metering-id ...] [flags]")
		assert.Contains(t, out, "--period")
		assert.Contains(t, out, "Global Flags:")
		assert.Contains(t, out, "--token")
		assert.NotContains(t, out, "[command]")
	})

	t.Run("a leaf command shows its Long text", func(t *testing.T) {
		out, err := execute(t, "thirdparty", "metering-points", "--help")
		require.NoError(t, err)
		assert.Contains(t, out, "Scope must be one of: authorizationId, customerCVR, customerKey")
		assert.Contains(t, out, "go-eloverblik thirdparty metering-points <scope> <identifier> [flags]")
	})

	t.Run("a leaf command shows its examples", func(t *testing.T) {
		out, err := execute(t, "customer", "timeseries", "--help")
		require.NoError(t, err)
		assert.Contains(t, out, "Examples:\n  # All of September, one value per day\n  go-eloverblik customer timeseries ")
	})

	t.Run("a group command lists its commands and the global flags", func(t *testing.T) {
		out, err := execute(t, "customer", "--help")
		require.NoError(t, err)
		assert.Contains(t, out, "go-eloverblik customer [command]")
		assert.Contains(t, out, "timeseries")
		assert.Contains(t, out, "Global Flags:")
		assert.Contains(t, out, "--token")
	})
}

// TestRootHelpCarriesTheRules checks that the root help states what an agent that has only
// the binary needs before its first call: how to pass the token, the rules that otherwise
// give a wrong result without an error, and where the full reference is. The root help is
// printed by rootHelpFunc, which has to print the Long text and the examples itself.
func TestRootHelpCarriesTheRules(t *testing.T) {
	out, err := execute(t, "--help")
	require.NoError(t, err)

	for _, rule := range []string{
		`--token "$ELO_TOKEN"`,
		"data access token",
		"[from, to)",
		"--to defaults to today",
		"730 days",
		"1 to 10 of them, each exactly 18 digits",
		"Results go to stdout as JSON",
		"go to stderr",
		"A metering point can fail on its own inside a successful response",
		"charge-links currently answers 404",
		"add-relation-by-code and delete-relation are retired",
		"https://github.com/slimcdk/go-eloverblik/blob/master/llms.md",
	} {
		assert.Contains(t, out, rule)
	}
	assert.Contains(t, out, "Examples:\n  export ELO_TOKEN=")
	assert.Less(t, strings.Index(out, "llms.md"), strings.Index(out, "Usage:"),
		"the Long text comes first, as in cobra's own help")
}

// leafCommands returns the commands below cmd that run something, hidden ones included,
// leaving out cobra's own help and completion commands.
func leafCommands(cmd *cobra.Command) []*cobra.Command {
	var leaves []*cobra.Command
	for _, sub := range cmd.Commands() {
		switch {
		case sub.Name() == "help" || sub.Name() == "completion":
		case sub.HasSubCommands():
			leaves = append(leaves, leafCommands(sub)...)
		default:
			leaves = append(leaves, sub)
		}
	}
	return leaves
}

// TestLeafCommandsDocumentThemselves checks that the help of every command tells how to
// call it, for an agent that has only the binary: a Long text saying what it calls and
// returns, and examples that pass the token. The retired commands are hidden and only
// fail, so they say that and have no examples.
func TestLeafCommandsDocumentThemselves(t *testing.T) {
	leaves := leafCommands(rootCmd)
	require.NotEmpty(t, leaves)

	for _, leaf := range leaves {
		t.Run(leaf.CommandPath(), func(t *testing.T) {
			assert.NotEmpty(t, leaf.Long)
			if !leaf.IsAvailableCommand() {
				assert.True(t, strings.HasPrefix(leaf.Long, "Retired."), "a hidden command is a retired one")
				assert.Empty(t, leaf.Example)
				return
			}
			assert.Contains(t, leaf.Example, `--token "$ELO_TOKEN"`)
		})
	}
}

// TestHelpUsesSyntheticMeteringPointIDs keeps real metering point IDs out of the help: an
// ID in a help text is 571313 followed by zeros and a two digit counter, which reads as a
// placeholder.
func TestHelpUsesSyntheticMeteringPointIDs(t *testing.T) {
	anyID := regexp.MustCompile(`\d{18}`)
	synthetic := regexp.MustCompile(`^5713130000000000\d\d$`)

	var check func(cmd *cobra.Command)
	check = func(cmd *cobra.Command) {
		texts := []string{cmd.Short, cmd.Long, cmd.Example}
		cmd.Flags().VisitAll(func(f *pflag.Flag) { texts = append(texts, f.Usage) })
		for _, text := range texts {
			for _, id := range anyID.FindAllString(text, -1) {
				assert.Regexp(t, synthetic, id, "in the help of %s", cmd.CommandPath())
			}
		}
		for _, sub := range cmd.Commands() {
			check(sub)
		}
	}
	check(rootCmd)
}

// TestCommandsThatNeedNoToken covers the commands that never use --token: they must run
// without it. The flag used to be marked required on the root, which cobra then demanded
// of every command, help and the shell completion scripts included.
func TestCommandsThatNeedNoToken(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		out, err := execute(t, "help")
		require.NoError(t, err)
		assert.Contains(t, out, "A CLI for the Danish Eloverblik platform")
	})

	t.Run("help for a command", func(t *testing.T) {
		out, err := execute(t, "help", "customer", "timeseries")
		require.NoError(t, err)
		assert.Contains(t, out, "go-eloverblik customer timeseries <metering-id>")
	})

	t.Run("completion", func(t *testing.T) {
		// cobra's completion command writes to the stdout it found when the first Execute
		// created it, a pipe an earlier test has closed. Remove it, so this Execute
		// creates it again while this test's stdout is being captured.
		for _, c := range rootCmd.Commands() {
			if c.Name() == "completion" {
				rootCmd.RemoveCommand(c)
			}
		}

		out, err := execute(t, "completion", "bash")
		require.NoError(t, err)
		assert.Contains(t, out, "bash completion")
	})
}

// TestCommandsRejectMissingToken covers the commands that use --token. cobra no longer
// checks for it, so each must refuse to run without one, and say why, before it builds a
// client.
//
// It must never reach Eloverblik, not even once that check is gone. The customer and
// thirdparty commands check the token in their group's PersistentPreRunE, which builds a
// real client once the check passes, and the command that runs next would send a request
// with it. So the test makes each of them fail instead of running. A fake client would
// not do: the PersistentPreRunE skips the check when a client is already set. The token
// command checks the token in its own RunE, and makes no request without --data-access,
// so it runs as it is.
func TestCommandsRejectMissingToken(t *testing.T) {
	saved := clientInstance
	t.Cleanup(func() { clientInstance = saved })

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"customer", []string{"customer", "details", "571313174002485069"}},
		{"thirdparty", []string{"thirdparty", "details", "571313174002485069"}},
		{"token", []string{"token"}},
		{"customer with an empty token", []string{"customer", "details", "571313174002485069", "--token", ""}},
		{"token with an empty token", []string{"token", "--token", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clientInstance = nil

			command, _, err := rootCmd.Find(tc.args)
			require.NoError(t, err)
			if command != tokenCmd {
				failIfRun(t, command)
			}

			_, err = execute(t, tc.args...)
			require.EqualError(t, err, `required flag "token" not set`)
			assert.Nil(t, clientInstance, "no client may be built without a token")
		})
	}
}

// failIfRun makes cmd return an error instead of running, until the test ends, so a
// command that got past a check it should have failed cannot send a request.
func failIfRun(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	run, runE := cmd.Run, cmd.RunE
	cmd.Run = nil
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return fmt.Errorf("%s ran, so it would have sent a request", c.CommandPath())
	}
	t.Cleanup(func() { cmd.Run, cmd.RunE = run, runE })
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
