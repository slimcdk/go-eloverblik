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
	Long: `Link 1 to 10 metering points to the user the refresh token belongs to, by metering point
ID. installations lists the metering points linked to the user.

Calls POST /meteringpoints/meteringpoint/relation/add on the Customer API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output: a JSON array with one object per metering point:
  {"result", "success", "errorCode", "errorText", "id", "stackTrace"}
A metering point that could not be linked has "success": false and the reason in
"errorCode" and "errorText", while the command still succeeds. "result" is left out when
the API sends none.

Linking a metering point with a web access code (add-relation-by-code) and deleting a
relation (delete-relation) are retired: Energinet retired both with DataHub 3.0.`,
	Example: `  go-eloverblik customer add-relation 571313000000000001 --token "$ELO_TOKEN"
  go-eloverblik customer add-relation 571313000000000001 571313000000000002 --token "$ELO_TOKEN" \
    | jq '.[] | select(.success == false) | {id, errorCode, errorText}'`,
	Args: meteringPointArgs,
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
// them fails with the reason. Without them, cobra would not report an unknown command below
// the root: it would print the customer help and exit 0. They no longer call the API,
// because the answer is already known.

var addRelationByCodeCmd = &cobra.Command{
	Use:   "add-relation-by-code <metering-id> <web-access-code>",
	Short: "Retired: linked a metering point to the authenticated user via a web access code",
	Long: `Retired. This command linked a metering point to the user with the metering point's web
access code, through PUT /meteringpoints/meteringpoint/relation/add/{id}/{code} on the
Customer API. Energinet retired web access codes and the endpoint with DataHub 3.0, and
the endpoint answers 410 Gone.

The command no longer calls the API: it fails with "endpoint retired by Energinet" and
exits with status 1. Data is shared through ElOverblik instead, see
https://docs.eloverblik.dk/docs/guides/data-sharing. To link a metering point by its ID,
use add-relation.`,
	Deprecated:   "Energinet retired web access codes with DataHub 3.0",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
	RunE: func(*cobra.Command, []string) error {
		return fmt.Errorf("%w: adding a metering point with a web access code is no longer supported, "+
			"see https://docs.eloverblik.dk/docs/guides/data-sharing for data sharing in ElOverblik", eloverblik.ErrorEndpointRetired)
	},
}

var deleteRelationCmd = &cobra.Command{
	Use:   "delete-relation <metering-id>",
	Short: "Retired: unlinked a metering point from the authenticated user",
	Long: `Retired. This command deleted the user's relation to a metering point, through
DELETE /meteringpoints/meteringpoint/relation/{id} on the Customer API. Energinet retired
the endpoint with DataHub 3.0, and it answers 410 Gone: a relation can no longer be
deleted.

The command no longer calls the API: it fails with "endpoint retired by Energinet" and
exits with status 1.`,
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
