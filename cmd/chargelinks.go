package cmd

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// newChargeLinksCmd builds a fresh command instance. It is added to both the customer and
// the thirdparty command, and cobra stores the parent on the command itself, so each
// parent needs its own instance.
func newChargeLinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "charge-links <metering-id> [metering-id ...]",
		Aliases: []string{"chargelinks"},
		Short:   "Get charge links with dated charge prices for one or more metering points",
		Long: "Get charge links with charges for one or more metering points.\n\n" +
			"Unlike 'charges', which only returns currently valid and future charges, this returns\n" +
			"the dated price series of every linked charge, so historic consumption can be priced.",
		Args: meteringPointArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			period, _ := cmd.Flags().GetString("period")

			// Check for mutual exclusivity and requirements
			if period != "" {
				if cmd.Flags().Changed("from") || cmd.Flags().Changed("to") {
					return errors.New("--period cannot be used with --from or --to")
				}
			} else {
				if !cmd.Flags().Changed("from") {
					return errors.New("either --period or --from is required")
				}
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			period, _ := cmd.Flags().GetString("period")
			fromFlag, _ := cmd.Flags().GetString("from")
			toFlag, _ := cmd.Flags().GetString("to")

			var from, to time.Time
			var err error

			if period != "" {
				from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.Period(period))
				cobra.CheckErr(err)
			} else {
				from, err = parseDate(fromFlag)
				cobra.CheckErr(err)
				to, err = parseDate(toFlag)
				cobra.CheckErr(err)
			}

			chargeLinks, err := clientInstance.GetChargeLinksWithCharges(args, from, to)
			cobra.CheckErr(err)

			bytes, err := json.Marshal(chargeLinks)
			cobra.CheckErr(err)
			_, err = output.Write(bytes)
			cobra.CheckErr(err)
		},
	}
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD, now, now-30d/w/m/y)")
	cmd.Flags().String("to", time.Now().Format(time.DateOnly), "end date (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "predefined period (yesterday, last_week, etc.)")
	return cmd
}

func init() {
	customerCmd.AddCommand(newChargeLinksCmd())
	thirdpartyCmd.AddCommand(newChargeLinksCmd())
}
