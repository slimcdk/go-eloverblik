package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var authorizationsCmd = &cobra.Command{
	Use:   "authorizations",
	Short: "Get authorizations (powers of attorney) granted by customers",
	Long: `List the authorizations (powers of attorney) customers have granted the third party the
token belongs to. Only valid or active authorizations are returned.

Calls GET /authorization/authorizations on the Third-Party API. Takes no arguments.

Output: a JSON array with one object per authorization:
  {"id", "thirdPartyName", "validFrom", "validTo", "customerName", "customerCVR",
   "customerKey", "includeFutureMeteringPoints", "timeStamp"}
"id", "customerCVR" and "customerKey" are the identifiers metering-points and
metering-point-ids take, with the scope authorizationId, customerCVR or customerKey.`,
	Example: `  go-eloverblik thirdparty authorizations --token "$ELO_TOKEN"

  # The customers and the identifiers to pass on
  go-eloverblik thirdparty authorizations --token "$ELO_TOKEN" \
    | jq '.[] | {customerName, id, customerCVR, customerKey, validTo}'`,
	Run: func(cmd *cobra.Command, args []string) {
		// Type assert to ThirdParty interface as GetAuthorizations is specific to the ThirdParty API
		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("the 'authorizations' command can only be used with the 'thirdparty' subcommand"))
		}

		authorizations, err := thirdpartyAPI.GetAuthorizations()
		cobra.CheckErr(err)

		bytes, err := json.Marshal(authorizations)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

func init() {
	thirdpartyCmd.AddCommand(authorizationsCmd)
}
