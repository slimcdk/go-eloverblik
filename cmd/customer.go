package cmd

import (
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var customerCmd = &cobra.Command{
	Use:   "customer",
	Short: "Commands for the Eloverblik Customer API",
	Long: `Commands for the Eloverblik Customer API, https://api.eloverblik.dk/customerapi/api: the
metering points of the person or company the refresh token belongs to.

Every command here requires --token, a Customer API refresh token created at
eloverblik.dk. Start with "installations" for the metering point IDs the other commands
take. See "go-eloverblik --help" for the date, ID and output rules all commands share.`,
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
