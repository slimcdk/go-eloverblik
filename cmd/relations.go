package cmd

import (
	"fmt"

	"github.com/drewstinnett/gout/v2"
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
		gout.MustPrint(results)
	},
}

var addRelationByCodeCmd = &cobra.Command{
	Use:   "add-relation-by-code <metering-id> <web-access-code>",
	Short: "Link a metering point to the authenticated user via a web access code",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("add-relation-by-code can only be used with the 'customer' subcommand"))
		}
		result, err := customerAPI.AddRelationByWebAccessCode(args[0], args[1])
		cobra.CheckErr(err)
		gout.MustPrint(result)
	},
}

var deleteRelationCmd = &cobra.Command{
	Use:   "delete-relation <metering-id>",
	Short: "Unlink a metering point from the authenticated user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("delete-relation can only be used with the 'customer' subcommand"))
		}
		ok, err := customerAPI.DeleteRelation(args[0])
		cobra.CheckErr(err)
		gout.MustPrint(ok)
	},
}

func init() {
	customerCmd.AddCommand(addRelationByIDCmd)
	customerCmd.AddCommand(addRelationByCodeCmd)
	customerCmd.AddCommand(deleteRelationCmd)
}
