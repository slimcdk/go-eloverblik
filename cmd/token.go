package cmd

import (
	"encoding/json"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Show what the Eloverblik token says about itself",
	Long: `Decode the claims of the token given with --token: which API and roles it was issued
for, who owns it, its name in the Eloverblik portal and when it expires. Takes no
arguments.

The claims are decoded, not verified, and no request is made to Eloverblik. Use
--data-access to exchange the refresh token for a data access token and decode that one
instead, which does make a request: GET /token on the API the refresh token's tokenType
names, one of the 2 /token calls a minute the API allows. The data access token itself is
never printed.

Output: one JSON object with "tokenType" and "expiresAt", and those of "tokenName",
"tokenId", "name", "subject", "company", "cvr", "userId", "thirdPartyId", "roles",
"loginType", "webApp", "issuer" and "audience" the token carries. "tokenType" names the
API and the kind of token, e.g. "THIRDPARTYAPI_Refresh" or "ThirdPartyApiDataAccess".
"expiresAt" is an RFC 3339 time in Copenhagen time, or "0001-01-01T00:00:00Z" when the
token carries no expiry. A token that cannot be decoded is an error.`,
	Example: `  # Which API is the token for, and when does it expire? Makes no request.
  go-eloverblik token --token "$ELO_TOKEN"
  go-eloverblik token --token "$ELO_TOKEN" | jq -r '.tokenType, .expiresAt'

  # Exchange it for a data access token and decode that one (one /token request)
  go-eloverblik token --data-access --token "$ELO_TOKEN"`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		token, err := refreshToken(cmd)
		if err != nil {
			return err
		}

		dataAccess, _ := cmd.Flags().GetBool("data-access")

		claims, err := eloverblik.ParseToken(token)
		if err != nil {
			return err
		}

		if dataAccess {
			// The token itself says which API it belongs to, so the right client can be
			// built without asking the user to repeat it.
			apiType, err := claims.APIType()
			if err != nil {
				return err
			}

			client := eloverblik.NewCustomer(token, clientOptions(cmd)...).(eloverblik.Client)
			if apiType == eloverblik.ThirdPartyApi {
				client = eloverblik.NewThirdParty(token, clientOptions(cmd)...)
			}

			if claims, err = client.DataAccessTokenClaims(); err != nil {
				return err
			}
		}

		bytes, err := json.Marshal(claims)
		if err != nil {
			return err
		}
		_, err = output.Write(bytes)
		return err
	},
}

func init() {
	tokenCmd.Flags().Bool("data-access", false, "Exchange the refresh token for a data access token and decode that instead")
	rootCmd.AddCommand(tokenCmd)
}
