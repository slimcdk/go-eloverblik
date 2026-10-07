package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var addRelationByIDCmd = &cobra.Command{
	Use:   "add-relation <metering-id> [metering-id ...]",
	Short: "Link one or more metering points to the authenticated user by ID",
	Args:  meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("add-relation can only be used with the 'customer' subcommand"))
		}
		results, err := customerAPI.AddRelationByID(args)
		cobra.CheckErr(err)
		bytes, err := json.Marshal(results)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

// Energinet retired the endpoints behind add-relation-by-code and delete-relation with
// DataHub 3.0; both answer 410 Gone. The commands stay, hidden, so a script that still runs
// them is told why instead of "unknown command". They no longer call the API, because the
// answer is already known.

var addRelationByCodeCmd = &cobra.Command{
	Use:          "add-relation-by-code <metering-id> <web-access-code>",
	Short:        "Link a metering point to the authenticated user via a web access code",
	Deprecated:   "Energinet retired web access codes with DataHub 3.0",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
	RunE: func(*cobra.Command, []string) error {
		return fmt.Errorf("%w: adding a metering point with a web access code is no longer supported, "+
			"see https://docs.eloverblik.dk/docs/guides/data-sharing for data sharing in ElOverblik", eloverblik.ErrorEndpointRetired)
	},
}

var deleteRelationCmd = &cobra.Command{
	Use:          "delete-relation <metering-id>",
	Short:        "Unlink a metering point from the authenticated user",
	Deprecated:   "Energinet retired the endpoint with DataHub 3.0",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(*cobra.Command, []string) error {
		return fmt.Errorf("%w: deleting a metering point relation is no longer supported", eloverblik.ErrorEndpointRetired)
	},
}

func init() {
	customerCmd.AddCommand(addRelationByIDCmd)
	customerCmd.AddCommand(addRelationByCodeCmd)
	customerCmd.AddCommand(deleteRelationCmd)
}
