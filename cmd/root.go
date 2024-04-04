package cmd

import (
	"os"

	"github.com/drewstinnett/gout/v2"
	"github.com/drewstinnett/gout/v2/formats/json"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var (
	ec eloverblik.Customer
)

var rootCmd = &cobra.Command{
	Use: "elob",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		token := cmd.Flag("token").Value.String()

		eloverblik.SetMode(eloverblik.ReleaseMode)

		var err error
		ec, err = eloverblik.CustomerClient(token)
		return err
	},

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

func init() {

	gout.SetFormatter(json.Formatter{})

	rootCmd.PersistentFlags().String("token", "", "Eloverblik Access Token (required)")
	rootCmd.MarkPersistentFlagRequired("token")
}
