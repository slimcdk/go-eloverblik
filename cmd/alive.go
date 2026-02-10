package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var aliveCmd = &cobra.Command{
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

func init() {
	customerCmd.AddCommand(aliveCmd)
	thirdpartyCmd.AddCommand(aliveCmd)
}
