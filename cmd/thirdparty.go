package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var thirdpartyCmd = &cobra.Command{
	Use:   "thirdparty",
	Short: "Commands for the Eloverblik Third-Party API",
	Long: `Commands for the Eloverblik Third-Party API, https://api.eloverblik.dk/thirdpartyapi/api:
the metering points customers have authorized the third party to read, through powers of
attorney.

Every command here requires --token, the third party's refresh token created at
eloverblik.dk. Start with "authorizations", then "metering-point-ids" for the metering
point IDs the other commands take. See "go-eloverblik --help" for the date, ID and output
rules all commands share.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if clientInstance != nil {
			return nil
		}
		token, err := refreshToken(cmd)
		if err != nil {
			return err
		}
		clientInstance = eloverblik.NewThirdParty(token, clientOptions(cmd)...)
		return nil
	},
}

var meteringPointsForScopeCmd = &cobra.Command{
	Use:   "metering-points <scope> <identifier>",
	Short: "Get metering points accessible under a specific authorization scope",
	Long: `List the metering points, with their address and master data, that the third party may
read under one authorization scope.

Calls GET /authorization/authorization/meteringpoints/{scope}/{identifier} on the
Third-Party API.

Arguments:
  <scope>       Scope must be one of: authorizationId, customerCVR, customerKey. It is
                sent as given, so spell it exactly so.
  <identifier>  The authorization's "id", the customer's "customerCVR" or the customer's
                "customerKey", as the authorizations command prints them.

Output: a JSON array with one object per metering point:
  {"meteringPointId", "typeOfMP", "accessFrom", "accessTo", "streetName",
   "buildingNumber", "postcode", "cityName", "meterNumber", "consumerCVR",
   "childMeteringPoints": [...], ...}
For the IDs alone, use metering-point-ids.`,
	Example: `  go-eloverblik thirdparty metering-points customerCVR 12345678 --token "$ELO_TOKEN"
  go-eloverblik thirdparty metering-points authorizationId "$AUTHORIZATION_ID" --token "$ELO_TOKEN"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		scope := args[0]
		identifier := args[1]

		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("metering-points can only be used with the 'thirdparty' subcommand"))
		}

		points, err := thirdpartyAPI.GetMeteringPointsForScope(eloverblik.AuthorizationScope(scope), identifier)
		cobra.CheckErr(err)
		bytes, err := json.Marshal(points)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

var meteringPointIDsForScopeCmd = &cobra.Command{
	Use:   "metering-point-ids <scope> <identifier>",
	Short: "Get metering point IDs accessible under a specific authorization scope",
	Long: `List the IDs of the metering points the third party may read under one authorization
scope: the IDs to pass to details, timeseries and charges.

Calls GET /authorization/authorization/meteringpointids/{scope}/{identifier} on the
Third-Party API.

Arguments:
  <scope>       Scope must be one of: authorizationId, customerCVR, customerKey. It is
                sent as given, so spell it exactly so.
  <identifier>  The authorization's "id", the customer's "customerCVR" or the customer's
                "customerKey", as the authorizations command prints them.

Output: a JSON array of metering point IDs, e.g.
  ["571313000000000001","571313000000000002"]`,
	Example: `  go-eloverblik thirdparty metering-point-ids customerCVR 12345678 --token "$ELO_TOKEN"
  go-eloverblik thirdparty metering-point-ids customerKey "$CUSTOMER_KEY" --token "$ELO_TOKEN"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		scope := args[0]
		identifier := args[1]

		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("metering-point-ids can only be used with the 'thirdparty' subcommand"))
		}

		ids, err := thirdpartyAPI.GetMeteringPointIDsForScope(eloverblik.AuthorizationScope(scope), identifier)
		cobra.CheckErr(err)
		bytes, err := json.Marshal(ids)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

func init() {
	thirdpartyCmd.AddCommand(meteringPointsForScopeCmd)
	thirdpartyCmd.AddCommand(meteringPointIDsForScopeCmd)
	rootCmd.AddCommand(thirdpartyCmd)
}
