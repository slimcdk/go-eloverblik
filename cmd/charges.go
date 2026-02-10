package cmd

import (
	"fmt"

	"github.com/drewstinnett/gout/v2"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var customerChargesCmd = &cobra.Command{
	Use:   "charges <metering-id> [metering-id ...]",
	Short: "Get charges (subscriptions, fees, tariffs) for one or more metering points",
	Args:  meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("charges can only be used with the 'customer' subcommand"))
		}
		charges, err := customerAPI.GetCustomerCharges(args)
		cobra.CheckErr(err)
		gout.MustPrint(charges)
	},
}

var thirdpartyChargesCmd = &cobra.Command{
	Use:   "charges <metering-id> [metering-id ...]",
	Short: "Get charges (subscriptions, tariffs) for one or more metering points",
	Args:  meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("charges can only be used with the 'thirdparty' subcommand"))
		}
		charges, err := thirdpartyAPI.GetThirdPartyCharges(args)
		cobra.CheckErr(err)
		gout.MustPrint(charges)
	},
}

var exportChargesCmd = &cobra.Command{
	Use:   "export-charges <metering-id> [metering-id ...]",
	Short: "Export charges (customer API only)",
	Args:  meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")

		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("export-charges can only be used with the 'customer' subcommand"))
		}

		stream, err := customerAPI.ExportCharges(args)
		cobra.CheckErr(err)

		err = outputStream(stream, format)
		cobra.CheckErr(err)
	},
}

func init() {
	customerCmd.AddCommand(customerChargesCmd)
	exportChargesCmd.Flags().String("format", "csv", "output format (csv, json)")
	customerCmd.AddCommand(exportChargesCmd)
	thirdpartyCmd.AddCommand(thirdpartyChargesCmd)
}
