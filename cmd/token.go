package cmd

import (
	"encoding/json"
	"fmt"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Show what the Eloverblik token says about itself",
	Long: `Decode the claims of the token given with --token: which API and roles it was issued
for, who owns it, its name in the Eloverblik portal and when it expires.

The claims are decoded, not verified, and no request is made to Eloverblik. Use
--data-access to exchange the refresh token for a data access token and decode that one
instead, which does make a request.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		token, err := cmd.Root().PersistentFlags().GetString("token")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("required flag \"token\" not set")
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
