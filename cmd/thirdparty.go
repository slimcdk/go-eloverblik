package cmd

import (
	"fmt"

	"github.com/drewstinnett/gout/v2"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var thirdpartyCmd = &cobra.Command{
	Use:   "thirdparty",
	Short: "Commands for the Eloverblik Third-Party API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		token, err := cmd.Root().PersistentFlags().GetString("token")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("required flag \"token\" not set")
		}
		clientInstance = eloverblik.NewThirdParty(token)
		return nil
	},
}

var meteringPointsForScopeCmd = &cobra.Command{
	Use:   "metering-points <scope> <identifier>",
	Short: "Get metering points accessible under a specific authorization scope",
	Long:  `Retrieve metering points for a given authorization scope and identifier. Scope must be one of: authorizationId, customerCVR, customerKey`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		scope := args[0]
		identifier := args[1]

		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("metering-points can only be used with the 'thirdparty' subcommand"))
		}

		points, err := thirdpartyAPI.GetMeteringPointsForScope(eloverblik.AuthorizationScope(scope), identifier)
		cobra.CheckErr(err)
		gout.MustPrint(points)
	},
}

var meteringPointIDsForScopeCmd = &cobra.Command{
	Use:   "metering-point-ids <scope> <identifier>",
	Short: "Get metering point IDs accessible under a specific authorization scope",
	Long:  `Retrieve metering point IDs for a given authorization scope and identifier. Scope must be one of: authorizationId, customerCVR, customerKey`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		scope := args[0]
		identifier := args[1]

		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("metering-point-ids can only be used with the 'thirdparty' subcommand"))
		}

		ids, err := thirdpartyAPI.GetMeteringPointIDsForScope(eloverblik.AuthorizationScope(scope), identifier)
		cobra.CheckErr(err)
		gout.MustPrint(ids)
	},
}

func init() {
	thirdpartyCmd.AddCommand(meteringPointsForScopeCmd)
	thirdpartyCmd.AddCommand(meteringPointIDsForScopeCmd)
	rootCmd.AddCommand(thirdpartyCmd)
}
