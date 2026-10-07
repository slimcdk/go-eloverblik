package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var installationsCmd = &cobra.Command{
	Use:   "installations",
	Short: "Get metering points (installations)",
	Long: `List the metering points of the user the refresh token belongs to, with their address and
master data: the metering point IDs the other customer commands take.

Calls GET /MeteringPoints/MeteringPoints?includeAll=<true|false> on the Customer API.
Takes no arguments.

Without --include-all (the default), only the metering points actively linked or related
to the user are returned. With --include-all, the list also holds the metering points
registered to the user's CPR or CVR number that are not linked.

Output: a JSON array with one object per metering point:
  {"meteringPointId", "typeOfMP", "streetName", "buildingNumber", "postcode", "cityName",
   "balanceSupplierName", "meterNumber", "consumerCVR", "consumerStartDate",
   "hasRelation", "isMovedOut", "childMeteringPoints": [...], ...}`,
	Example: `  go-eloverblik customer installations --token "$ELO_TOKEN"
  go-eloverblik customer installations --include-all --token "$ELO_TOKEN"

  # The IDs alone
  go-eloverblik customer installations --token "$ELO_TOKEN" | jq -r '.[].meteringPointId'`,
	Run: func(cmd *cobra.Command, args []string) {

		includeAll, _ := cmd.Flags().GetBool("include-all")

		// Type assert to Customer interface as GetMeteringPoints is specific to the Customer API
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("the 'installations' command can only be used with the 'customer' subcommand"))
		}

		meters, err := customerAPI.GetMeteringPoints(includeAll)
		cobra.CheckErr(err)

		bytes, err := json.Marshal(meters)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

func init() {
	installationsCmd.Flags().Bool("include-all", false, "also list the metering points registered to the user's CPR or CVR number that are not linked")
	customerCmd.AddCommand(installationsCmd)
}
