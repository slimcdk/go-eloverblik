package cmd

import (
	"os"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// clientInstance will hold the instantiated client (either Customer or ThirdParty)
var clientInstance eloverblik.Client

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

func init() {
	rootCmd.PersistentFlags().String("token", "", "Eloverblik Access Token (required)")
	_ = rootCmd.MarkPersistentFlagRequired("token")
}
