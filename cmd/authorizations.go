package cmd

import (
	"fmt"

	"github.com/drewstinnett/gout/v2"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var authorizationsCmd = &cobra.Command{
	Use:   "authorizations",
	Short: "Get authorizations (powers of attorney) granted by customers",
	Long:  `This request is used for retrieving details about authorizations (powers of attorney) granted by customers. Only data regarding valid or active authorizations is returned.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Type assert to ThirdParty interface as GetAuthorizations is specific to the ThirdParty API
		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("the 'authorizations' command can only be used with the 'thirdparty' subcommand"))
		}

		authorizations, err := thirdpartyAPI.GetAuthorizations()
		cobra.CheckErr(err)

		gout.MustPrint(authorizations)
	},
}

func init() {
	thirdpartyCmd.AddCommand(authorizationsCmd)
}
