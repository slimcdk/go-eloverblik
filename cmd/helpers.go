package cmd

import (
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// clientOptions builds the library options from the persistent flags on the root command.
func clientOptions(cmd *cobra.Command) []eloverblik.Option {
	opts := make([]eloverblik.Option, 0, 1)

	if printHeaders, err := cmd.Root().PersistentFlags().GetBool("print-response-headers"); err == nil && printHeaders {
		opts = append(opts, eloverblik.WithResponseHeaderOutput(headerOutput))
	}

	return opts
}
