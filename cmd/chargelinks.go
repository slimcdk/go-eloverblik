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
		Short:   "Get charge links with dated charge prices (404 while the feature is disabled)",
		Long: `Get the charge links of 1 to 10 metering points in the half-open range [from, to), with
the dated price series of every charge they link to.

NOT AVAILABLE YET. Both of Energinet's OpenAPI documents declare
getchargelinkswithcharges, and document a 404 from it as "When the Charges integration
feature is disabled". That is what the live API answered when checked on 2026-07-13 with
valid Customer and Third-Party tokens: 404 Not Found on BOTH the Customer API and the
Third-Party API, on every documented path, while 'charges' answered 200 with the same
tokens. Until Energinet enables the feature, every call returns 404.

What to use today: 'charges'. It returns the subscriptions and tariffs of a metering point
(on the Customer API also the fees), but only those that are currently valid or take
effect in the future, so it cannot price consumption that already happened.

Calls POST /meteringpoints/meteringpoint/getchargelinkswithcharges on the Customer API, or
POST /meteringpoint/getchargelinkswithcharges on the Third-Party API, as both documents
specify it, with one difference: the API takes an interval per metering point, and this
client applies the one interval given here to all of them.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Dates: give either --period, or --from with an optional --to, as for timeseries, but they
are sent as timestamps in Copenhagen time, not as dates. A date in --from or --to is a
Danish day, so 2026-07-01 means midnight in Copenhagen, sent as 2026-07-01T00:00:00+02:00,
and the interval stops before --to. now and now-30d keep the time of day, and a this_*
period runs up to now, today included.

What it will return: the dated price series of every charge a metering point is linked
to, the charge link periods and their factors, the VAT classification and the tax
indicator, so historic consumption can be priced. As JSON, one object:
  {"results": [{"meteringPointId", "error",
                "chargeLinks": [{"meteringPointId",
                                 "chargeIdentifier": {"code", "owner", "type"},
                                 "chargeLinkPeriods": [{"factor", "from", "to"}]}]}],
   "chargeInformations": [{"chargeIdentifier", "taxIndicator", "resolution",
                           "pricingCategory", "chargeInformationPeriods": [{"name",
                           "description", "transparentInvoicing", "from", "to",
                           "vatClassification"}],
                           "chargeSeriesPoints": [{"from", "to", "price"}]}]}
A metering point that failed has the reason in its "error", and the others still resolve.
A "to" that is null is open ended.`,
		Example: `  go-eloverblik customer charge-links 571313000000000001 --period last_month --token "$ELO_TOKEN"
  go-eloverblik thirdparty charge-links 571313000000000001 571313000000000002 --from 2026-09-01 --to 2026-10-01 --token "$ELO_TOKEN"`,
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
	cmd.Flags().String("from", "", "start date, inclusive (YYYY-MM-DD, now, now-30d/w/m/y); required unless --period is given")
	cmd.Flags().String("to", today(), "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "named range instead of --from and --to: yesterday, this_week, last_week, this_month, last_month, this_year or last_year")
	return cmd
}

func init() {
	customerCmd.AddCommand(newChargeLinksCmd())
	thirdpartyCmd.AddCommand(newChargeLinksCmd())
}
