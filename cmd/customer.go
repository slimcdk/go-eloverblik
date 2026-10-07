package cmd

import (
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var customerCmd = &cobra.Command{
	Use:   "customer",
	Short: "Commands for the Eloverblik Customer API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if clientInstance != nil {
			return nil
		}
		token, err := refreshToken(cmd)
		if err != nil {
			return err
		}
		clientInstance = eloverblik.NewCustomer(token, clientOptions(cmd)...)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(customerCmd)
}
