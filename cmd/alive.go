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
		Long: `Check whether the API is up: call GET /isalive on the API of the parent command, the
Customer API for "customer alive" and the Third-Party API for "thirdparty alive". Takes
no arguments.

Output: one line of text on stdout, not JSON:
  API is alive and operational     the API answered with a 2xx status
  API is not responding normally   it answered with any other status
Both exit with status 0. Only a request that gets no answer, e.g. on a network error,
fails the command, with the error on stderr and status 1.

The request carries no token, but --token is still required, as for every customer and
thirdparty command.`,
		Example: `  go-eloverblik customer alive --token "$ELO_TOKEN"
  go-eloverblik thirdparty alive --token "$ELO_TOKEN"`,
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
