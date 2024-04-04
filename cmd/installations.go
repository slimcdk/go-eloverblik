package cmd

import (
	"github.com/drewstinnett/gout/v2"
	"github.com/spf13/cobra"
)

// go run main.go --token=... installations [include-all=true] | jq
var installationsCmd = &cobra.Command{
	Use: "installations",
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(gout.BindCobra(cmd, nil))

		includeAll, _ := cmd.Flags().GetBool("include-all")

		meters, err := ec.GetMeteringPoints(includeAll)
		cobra.CheckErr(err)

		gout.MustPrint(meters)
	},
}

func init() {
	installationsCmd.Flags().Bool("include-all", true, "Include all")

	rootCmd.AddCommand(installationsCmd)
}
