package cmd

import (
	"fmt"

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
		token, err := cmd.Root().PersistentFlags().GetString("token")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("required flag \"token\" not set")
		}
		clientInstance = eloverblik.NewCustomer(token)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(customerCmd)
}
