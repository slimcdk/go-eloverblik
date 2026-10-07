package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// clientInstance will hold the instantiated client (either Customer or ThirdParty)
var clientInstance eloverblik.Client

// headerOutput is the destination for the HTTP response headers printed with
// --print-response-headers (configurable for testing). It defaults to stderr so
// stdout stays clean, parseable JSON.
var headerOutput io.Writer = os.Stderr

var rootCmd = &cobra.Command{
	Use:   "go-eloverblik",
	Short: "A CLI for the Danish Eloverblik platform",
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// write data access token to temporary location for reuse
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// refreshToken returns the refresh token given with --token, or an error when none was
// given. The commands that use the token call it instead of having cobra require the
// flag, which cobra would then demand of every command, help and completion included.
func refreshToken(cmd *cobra.Command) (string, error) {
	token, err := cmd.Root().PersistentFlags().GetString("token")
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", errors.New(`required flag "token" not set`)
	}
	return token, nil
}

// helpFunc returns the help for the whole command tree. A help function set on the root
// is inherited by every command below it, so it has to pick: the root lists each group's
// commands under the group's name, and every other command gets cobra's default help,
// which is what tells how to call it: the usage line with its arguments, the Long text,
// the examples, its flags and the global flags such as --token.
func helpFunc(defaultHelp func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.HasParent() {
			defaultHelp(cmd, args)
			return
		}
		rootHelpFunc(cmd, args)
	}
}

// rootHelpFunc prints the root help: the commands of each group listed under the group's
// name, then the root's own flags.
func rootHelpFunc(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	printf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, format, args...)
	}

	printf("%s\n\nUsage:\n  %s [command]\n\nAvailable Commands:\n", cmd.Short, cmd.CommandPath())
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.Name() == "help" {
			continue
		}
		if sub.HasAvailableSubCommands() {
			printf("\n  %s\n", sub.Name())
			for _, subsub := range sub.Commands() {
				if !subsub.IsAvailableCommand() {
					continue
				}
				printf("    %-24s %s\n", subsub.Name(), subsub.Short)
			}
		} else {
			printf("  %-26s %s\n", sub.Name(), sub.Short)
		}
	}
	if flags := cmd.LocalFlags().FlagUsages(); flags != "" {
		printf("\nFlags:\n%s", flags)
	}
	printf("\nUse \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

func init() {
	// --token is not marked required: the commands that use it check it with refreshToken.
	rootCmd.PersistentFlags().String("token", "", "Eloverblik refresh token (required by the customer, thirdparty and token commands)")
	rootCmd.PersistentFlags().Bool("print-response-headers", false, "Print HTTP response headers from the Eloverblik API to stderr")
	rootCmd.SetHelpFunc(helpFunc(rootCmd.HelpFunc()))
}
