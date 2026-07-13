package cmd

import (
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

func rootHelpFunc(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s\n\nUsage:\n  %s [command]\n\nAvailable Commands:\n", cmd.Short, cmd.CommandPath())
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.Name() == "help" {
			continue
		}
		if sub.HasAvailableSubCommands() {
			fmt.Fprintf(w, "\n  %s\n", sub.Name())
			for _, subsub := range sub.Commands() {
				if !subsub.IsAvailableCommand() {
					continue
				}
				fmt.Fprintf(w, "    %-24s %s\n", subsub.Name(), subsub.Short)
			}
		} else {
			fmt.Fprintf(w, "  %-26s %s\n", sub.Name(), sub.Short)
		}
	}
	if flags := cmd.LocalFlags().FlagUsages(); flags != "" {
		fmt.Fprintf(w, "\nFlags:\n%s", flags)
	}
	fmt.Fprintf(w, "\nUse \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

func init() {
	rootCmd.PersistentFlags().String("token", "", "Eloverblik Access Token (required)")
	_ = rootCmd.MarkPersistentFlagRequired("token")
	rootCmd.PersistentFlags().Bool("print-response-headers", false, "Print HTTP response headers from the Eloverblik API to stderr")
	rootCmd.SetHelpFunc(rootHelpFunc)
}
