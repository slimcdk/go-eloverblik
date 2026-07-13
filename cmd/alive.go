package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newAliveCmd builds a fresh "alive" command for a single parent. cobra stores the
// parent on the command itself, so one shared instance added to both "customer" and
// "thirdparty" would keep only the parent it was added to last: "customer alive" would
// then run the ThirdParty PersistentPreRunE and probe the ThirdParty API.
func newAliveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alive",
		Short: "Check if the API is operational",
		Run: func(cmd *cobra.Command, args []string) {
			alive, err := clientInstance.IsAlive()
			cobra.CheckErr(err)

			if alive {
				fmt.Println("API is alive and operational")
			} else {
				fmt.Println("API is not responding normally")
			}
		},
	}
}

func init() {
	customerCmd.AddCommand(newAliveCmd())
	thirdpartyCmd.AddCommand(newAliveCmd())
}
