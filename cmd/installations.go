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
	Long: `This request is used for retrieving a list of metering points associated with a specific user.
If the parameter 'include-all' is 'false' (default), only metering points actively linked or related to the user are returned.
If 'include-all' is 'true', the list is merged with additional non-linked metering points registered to the CPR or CVR of the user.`,
	Run: func(cmd *cobra.Command, args []string) {

		includeAll, _ := cmd.Flags().GetBool("include-all")

		// Type assert to Customer interface as GetMeteringPoints is a specific to the Customer API
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
	installationsCmd.Flags().Bool("include-all", false, "Include metering points not actively linked to the user")
	customerCmd.AddCommand(installationsCmd)
}
